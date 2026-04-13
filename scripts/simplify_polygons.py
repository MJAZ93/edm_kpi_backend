#!/usr/bin/env python3
"""
Simplify ASC and Regiao polygon coordinates stored in PostgreSQL.

Problem: Polygon data has ~163k points with 14 decimal places → ~6.3 MB for ASCs alone.
Solution: Douglas-Peucker simplification (~1 point/km) + precision reduction (6 decimals).

Usage:
  # Preview — see before/after stats, no DB changes
  python3 scripts/simplify_polygons.py --preview

  # Generate SQL file
  python3 scripts/simplify_polygons.py --output simplify.sql

  # Apply directly to DB (reads DB_* env vars or .env)
  python3 scripts/simplify_polygons.py --apply

  # Custom tolerance (default 0.008 ≈ ~1km at equator)
  python3 scripts/simplify_polygons.py --tolerance 0.005 --output simplify.sql

Requires: pip install shapely psycopg2-binary python-dotenv
"""

import argparse
import json
import os
import sys
from pathlib import Path

try:
    from shapely.geometry import shape, mapping
    from shapely import __version__ as shapely_version
except ImportError:
    print("ERROR: shapely is required. Install with: pip install shapely")
    sys.exit(1)

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

# ~1 km tolerance in degrees (1° ≈ 111 km → 1 km ≈ 0.009°)
DEFAULT_TOLERANCE = 0.008

# 6 decimal places ≈ 0.11 m precision (more than enough for ASC boundaries)
COORD_PRECISION = 6


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def round_coords(geom_dict: dict, precision: int = COORD_PRECISION) -> dict:
    """Round all coordinates in a GeoJSON geometry dict to N decimal places."""
    def _round(coords):
        if isinstance(coords[0], (list, tuple)):
            return [_round(c) for c in coords]
        return [round(c, precision) for c in coords]

    result = dict(geom_dict)
    result['coordinates'] = _round(result['coordinates'])
    return result


def count_points(geom_dict: dict) -> int:
    """Count total coordinate points in a GeoJSON geometry."""
    def _count(coords):
        if isinstance(coords[0], (list, tuple)):
            return sum(_count(c) for c in coords)
        return 1
    return _count(geom_dict.get('coordinates', []))


def simplify_geojson(geojson_str: str, tolerance: float = DEFAULT_TOLERANCE) -> tuple[str, int, int]:
    """
    Simplify a GeoJSON geometry string.
    Returns: (simplified_json_str, original_points, new_points)
    """
    try:
        geom_dict = json.loads(geojson_str)
    except (json.JSONDecodeError, TypeError):
        return geojson_str, 0, 0

    if 'type' not in geom_dict or 'coordinates' not in geom_dict:
        return geojson_str, 0, 0

    original_pts = count_points(geom_dict)

    # Convert to Shapely, simplify, convert back
    geom = shape(geom_dict)
    if geom.is_empty:
        return geojson_str, original_pts, original_pts

    simplified = geom.simplify(tolerance, preserve_topology=True)
    result = mapping(simplified)

    # Round coordinates to reduce decimal precision
    result = round_coords(result)

    new_pts = count_points(result)

    # Compact JSON (no extra whitespace)
    result_str = json.dumps(result, separators=(',', ':'))

    return result_str, original_pts, new_pts


def load_env():
    """Load .env file for DB credentials."""
    env_path = Path(__file__).resolve().parent.parent / '.env'
    if not env_path.exists():
        return
    try:
        from dotenv import load_dotenv
        load_dotenv(env_path)
    except ImportError:
        # Manual fallback: parse KEY=VALUE lines
        with open(env_path) as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith('#') or '=' not in line:
                    continue
                key, _, val = line.partition('=')
                key = key.strip()
                val = val.strip().strip('"').strip("'")
                os.environ.setdefault(key, val)
    print(f"  Loaded .env from {env_path}")


def get_db_connection():
    """Create psycopg2 connection from env vars."""
    try:
        import psycopg2
    except ImportError:
        print("ERROR: psycopg2 is required for --apply. Install with: pip install psycopg2-binary")
        sys.exit(1)

    load_env()

    conn = psycopg2.connect(
        host=os.getenv('DB_HOST', 'localhost'),
        port=int(os.getenv('DB_PORT', '5432')),
        user=os.getenv('DB_USER', 'postgres'),
        password=os.getenv('DB_PASSWORD', ''),
        dbname=os.getenv('DB_NAME', 'edm_kpi'),
    )
    return conn


def fetch_polygons(conn, table: str) -> list[tuple[int, str, str]]:
    """Fetch (id, name, polygon) rows that have non-empty polygon data."""
    cur = conn.cursor()
    cur.execute(f"""
        SELECT id, name, polygon
        FROM {table}
        WHERE polygon IS NOT NULL AND polygon != '' AND deleted_at IS NULL
        ORDER BY id
    """)
    rows = cur.fetchall()
    cur.close()
    return rows


def escape_sql_string(s: str) -> str:
    """Escape single quotes for SQL."""
    return s.replace("'", "''")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description='Simplify ASC/Regiao polygon coordinates')
    parser.add_argument('--tolerance', type=float, default=DEFAULT_TOLERANCE,
                        help=f'Douglas-Peucker tolerance in degrees (default: {DEFAULT_TOLERANCE} ≈ 1km)')
    parser.add_argument('--preview', action='store_true',
                        help='Preview stats without making changes')
    parser.add_argument('--output', type=str, default=None,
                        help='Write SQL UPDATE statements to file')
    parser.add_argument('--apply', action='store_true',
                        help='Apply directly to database')
    parser.add_argument('--tables', nargs='+', default=['ascs', 'regiaos'],
                        help='Tables to process (default: ascs regiaos)')

    args = parser.parse_args()

    if not args.preview and not args.output and not args.apply:
        print("Specify --preview, --output <file>, or --apply")
        parser.print_help()
        sys.exit(1)

    print(f"╔══════════════════════════════════════════════════════════════╗")
    print(f"║  Polygon Simplifier — tolerance={args.tolerance}° (~{args.tolerance * 111:.1f} km)  ║")
    print(f"║  Coordinate precision: {COORD_PRECISION} decimals (~0.11 m)              ║")
    print(f"╚══════════════════════════════════════════════════════════════╝\n")

    conn = get_db_connection()
    sql_statements = []

    total_before_pts = 0
    total_after_pts = 0
    total_before_bytes = 0
    total_after_bytes = 0

    for table in args.tables:
        print(f"── {table.upper()} ──────────────────────────────────────────────")
        rows = fetch_polygons(conn, table)
        print(f"  Found {len(rows)} rows with polygon data\n")

        for row_id, name, polygon_str in rows:
            before_bytes = len(polygon_str.encode('utf-8'))
            simplified, before_pts, after_pts = simplify_geojson(polygon_str, args.tolerance)
            after_bytes = len(simplified.encode('utf-8'))

            total_before_pts += before_pts
            total_after_pts += after_pts
            total_before_bytes += before_bytes
            total_after_bytes += after_bytes

            reduction_pct = ((before_pts - after_pts) / before_pts * 100) if before_pts > 0 else 0
            size_pct = ((before_bytes - after_bytes) / before_bytes * 100) if before_bytes > 0 else 0

            print(f"  [{row_id:3d}] {name[:35]:35s}  "
                  f"{before_pts:6,d} → {after_pts:5,d} pts ({reduction_pct:5.1f}% fewer)  "
                  f"{before_bytes/1024:7.1f} → {after_bytes/1024:6.1f} KB ({size_pct:4.1f}% smaller)")

            if before_pts != after_pts or before_bytes != after_bytes:
                sql = f"UPDATE {table} SET polygon = '{escape_sql_string(simplified)}' WHERE id = {row_id};"
                sql_statements.append(sql)

        print()

    # Summary
    pts_reduction = ((total_before_pts - total_after_pts) / total_before_pts * 100) if total_before_pts > 0 else 0
    size_reduction = ((total_before_bytes - total_after_bytes) / total_before_bytes * 100) if total_before_bytes > 0 else 0

    print(f"═══════════════════════════════════════════════════════════════")
    print(f"  TOTAL POINTS:  {total_before_pts:,d} → {total_after_pts:,d}  ({pts_reduction:.1f}% reduction)")
    print(f"  TOTAL SIZE:    {total_before_bytes/1024/1024:.2f} MB → {total_after_bytes/1024/1024:.2f} MB  ({size_reduction:.1f}% smaller)")
    print(f"  SQL UPDATES:   {len(sql_statements)} statements")
    print(f"═══════════════════════════════════════════════════════════════\n")

    if args.preview:
        print("  (--preview mode, no changes applied)")

    if args.output and sql_statements:
        output_path = Path(args.output)
        with open(output_path, 'w') as f:
            f.write("-- Auto-generated polygon simplification\n")
            f.write(f"-- Tolerance: {args.tolerance}° (~{args.tolerance * 111:.1f} km)\n")
            f.write(f"-- Precision: {COORD_PRECISION} decimal places\n")
            f.write(f"-- Points:    {total_before_pts:,d} → {total_after_pts:,d}\n")
            f.write(f"-- Size:      {total_before_bytes/1024/1024:.2f} MB → {total_after_bytes/1024/1024:.2f} MB\n\n")
            f.write("BEGIN;\n\n")
            for stmt in sql_statements:
                f.write(stmt + "\n")
            f.write("\nCOMMIT;\n")

        file_size = output_path.stat().st_size
        print(f"  ✓ Written to {output_path} ({file_size/1024:.1f} KB)")

    if args.apply and sql_statements:
        print("  Applying to database...")
        cur = conn.cursor()
        try:
            for stmt in sql_statements:
                cur.execute(stmt)
            conn.commit()
            print(f"  ✓ Applied {len(sql_statements)} updates successfully!")
        except Exception as e:
            conn.rollback()
            print(f"  ✗ ERROR: {e}")
            sys.exit(1)
        finally:
            cur.close()

    conn.close()


if __name__ == '__main__':
    main()
