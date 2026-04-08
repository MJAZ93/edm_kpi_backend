#!/usr/bin/env python3
"""
Seed script — cleans the DB (keeps admin user) then inserts:
  - 5 Regiões
  - 25 ASCs with their polygons from asc_coordinates.json
  - Region polygons computed as the union of their ASC polygons
"""

import json, os, sys
import psycopg2
from shapely.geometry import Polygon, MultiPolygon
from shapely.ops import unary_union

# ── DB connection ─────────────────────────────────────────────────────────────
DB = dict(
    host=os.getenv("DB_HOST", "localhost"),
    port=int(os.getenv("DB_PORT", 5432)),
    user=os.getenv("DB_USER", "postgres"),
    password=os.getenv("DB_PASSWORD", "postgres"),
    dbname=os.getenv("DB_NAME", "kpi_db"),
)

# ── Load coordinate file ──────────────────────────────────────────────────────
COORD_FILE = os.getenv(
    "COORD_FILE",
    "/Users/afonso.junior/Downloads/asc_coordinates.json",
)
with open(COORD_FILE) as f:
    raw = json.load(f)           # list of {NAME: [[lat,lng], ...]}

# Build lookup: normalised-name → list of [lat, lng]
coord_map: dict[str, list] = {}
for item in raw:
    for name, coords in item.items():
        key = name.strip().upper()
        if key not in coord_map:           # keep first occurrence for duplicates
            coord_map[key] = coords

# ── ASC catalogue ─────────────────────────────────────────────────────────────
# (display_name, json_key, region_code)
ASCS = [
    ("ASC Angoche",       "ASC ANGOCHE",       "DRN"),
    ("ASC Matola",        "ASC MATOLA",         "DRS"),
    ("ASC Mocuba",        "ASC MOCUBA",         "DRC"),
    ("ASC Chimoio",       "ASC CHIMOIO",        "DRC"),
    ("ASC Pemba",         "ASC PEMBA",          "DRN"),
    ("ASC Kamubukwana",   "ASC KAMUBUKWANA",    "DRCM"),
    ("ASC Kamaxakene",    "ASC KAMAXAQUENI",    "DRCM"),   # same polygon
    ("ASC Quelimane",     "ASC QUELIMANE",      "DRC"),
    ("ASC Kamavota",      "ASC KAMAVOTA",       "DRCM"),
    ("ASC Kapfumo",       "ASC KAMPFUMO",       "DRCM"),
    ("ASC Nacala",        "ASC NACALA",         "DRN"),
    ("ASC Nampula",       "ASC NAMPULA",        "DRN"),
    ("ASC Machava",       "ASC MACHAVA",        "DRS"),
    ("ASC Infulene",      "ASC INFULENE",       "DRS"),
    ("ASC da Beira",      "ASC BEIRA",          "DRC"),
    ("ASC Tete",          "ASC TETE",           "DRC"),
    ("ASC Kaguava",       "ASC KAGUAVA",        "DRCM"),
    ("ASC Caia",          "ASC CAIA",           "DRC"),
    ("ASC Lichinga",      "ASC LICHINGA",       "DRN"),
    ("ASC Vilanculo",     "ASC DE VILANKULOS",  "DRS"),
    ("ASC Inhambane",     "ASC INHAMBANE",      "DRS"),
    ("ASC Chokwe",        "ASC CHOKWE",         "DRS"),
    ("ASC Xai Xai",       "ASC XAI-XAI",        "DRS"),
    ("ASC Kamaxaquene",   "ASC KAMAXAQUENI",    "DRCM"),   # same polygon
    ("ASC Boane",         "ASC BOANE",          "DRS"),
]

# ── Region catalogue ──────────────────────────────────────────────────────────
REGIOES = [
    ("Direcção Regional Norte",            "DRN"),
    ("Direcção Regional Centro",           "DRC"),
    ("Direcção Regional Sul",              "DRS"),
    ("Direcção Regional Cidade de Maputo", "DRCM"),
    ("Direcção Regional Província de Maputo", "DRPM"),
]

# ── Helpers ───────────────────────────────────────────────────────────────────

def coords_to_geojson_polygon(coords_latlon: list) -> str:
    """coords are [lat, lon] → GeoJSON wants [lon, lat]."""
    ring = [[lon, lat] for lat, lon in coords_latlon]
    if ring[0] != ring[-1]:
        ring.append(ring[0])
    return json.dumps({"type": "Polygon", "coordinates": [ring]})


def coords_to_shapely(coords_latlon: list) -> Polygon:
    pts = [(lon, lat) for lat, lon in coords_latlon]
    if len(pts) < 3:
        return None
    poly = Polygon(pts)
    if not poly.is_valid:
        poly = poly.buffer(0)
    return poly


def shape_to_geojson(geom) -> str:
    """Convert Shapely geometry to GeoJSON string."""
    from shapely.geometry import mapping
    return json.dumps(mapping(geom))


# ── Main ──────────────────────────────────────────────────────────────────────

def main():
    conn = psycopg2.connect(**DB)
    cur = conn.cursor()

    print("── Cleaning database (keeping admin user) ──────────────────────────")
    tables_to_truncate = [
        "milestone_progresses",
        "milestones",
        "blockers",
        "task_scopes",
        "tasks",
        "project_direcoes",
        "projects",
        "performance_caches",
        "notifications",
        "audit_logs",
        "ascs",
        "regiaos",
        "departamento_users",
        "departamentos",
        "direcaos",
        "pelouros",
    ]
    for t in tables_to_truncate:
        cur.execute(f'TRUNCATE TABLE "{t}" RESTART IDENTITY CASCADE')
        print(f"  truncated {t}")

    # Delete non-admin users (keep CA and ADMIN)
    cur.execute("DELETE FROM users WHERE role NOT IN ('CA', 'ADMIN')")
    print("  deleted non-admin/non-CA users")

    conn.commit()

    # ── Insert Regiões ────────────────────────────────────────────────────────
    print("\n── Inserting Regiões ──────────────────────────────────────────────")
    regiao_ids: dict[str, int] = {}
    for name, code in REGIOES:
        cur.execute(
            """INSERT INTO regiaos (name, code, created_at, updated_at)
               VALUES (%s, %s, NOW(), NOW()) RETURNING id""",
            (name, code),
        )
        rid = cur.fetchone()[0]
        regiao_ids[code] = rid
        print(f"  [{rid}] {name} ({code})")
    conn.commit()

    # ── Insert ASCs ───────────────────────────────────────────────────────────
    print("\n── Inserting ASCs ──────────────────────────────────────────────────")
    asc_shapes: dict[str, list] = {code: [] for _, code in REGIOES}

    missing = []
    for display_name, json_key, r_code in ASCS:
        coords = coord_map.get(json_key)
        polygon_str = None
        shape = None
        if coords:
            polygon_str = coords_to_geojson_polygon(coords)
            shape = coords_to_shapely(coords)
            if shape and shape.is_valid:
                asc_shapes[r_code].append(shape)
        else:
            missing.append(json_key)

        regiao_id = regiao_ids.get(r_code)
        cur.execute(
            """INSERT INTO ascs (name, code, regiao_id, polygon, created_at, updated_at)
               VALUES (%s, %s, %s, %s, NOW(), NOW()) RETURNING id""",
            (display_name, r_code, regiao_id, polygon_str),
        )
        aid = cur.fetchone()[0]
        geo_flag = "✓" if polygon_str else "✗ NO POLYGON"
        print(f"  [{aid}] {display_name} → {r_code}  {geo_flag}")

    if missing:
        print(f"\n  WARNING — no coordinates found for: {missing}")

    conn.commit()

    # ── Compute & update Região polygons ──────────────────────────────────────
    print("\n── Computing region polygons (union of ASCs) ────────────────────")
    for _, code in REGIOES:
        shapes = asc_shapes.get(code, [])
        if not shapes:
            print(f"  {code}: no ASC shapes — skipping")
            continue
        union = unary_union(shapes)
        if not union.is_valid:
            union = union.buffer(0)
        geojson_str = shape_to_geojson(union)
        cur.execute(
            "UPDATE regiaos SET polygon = %s WHERE code = %s",
            (geojson_str, code),
        )
        pts = union.exterior.coords.__len__() if hasattr(union, 'exterior') else "multi"
        print(f"  {code}: polygon computed ({type(union).__name__})")

    conn.commit()
    cur.close()
    conn.close()

    print("\n✅  Seed complete.")


if __name__ == "__main__":
    main()
