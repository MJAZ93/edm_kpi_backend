package util

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

func SendEmail(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	if host == "" {
		log.Printf("[EMAIL-SKIP] SMTP not configured. To: %s, Subject: %s", to, subject)
		return nil
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	auth := smtp.PlainAuth("", user, pass, host)
	addr := fmt.Sprintf("%s:%s", host, port)

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		log.Printf("[EMAIL-ERROR] Failed to send to %s: %v", to, err)
		return err
	}

	log.Printf("[EMAIL-SENT] To: %s, Subject: %s", to, subject)
	return nil
}

func EmailPasswordReset(to, name, token string) {
	scheme := os.Getenv("SCHEME")
	ip := os.Getenv("IP")
	port := os.Getenv("PORT")
	link := fmt.Sprintf("%s://%s:%s/reset-password?token=%s", scheme, ip, port, token)

	body := fmt.Sprintf(`
		<h2>Redefinição de Password</h2>
		<p>Olá %s,</p>
		<p>Recebemos um pedido para redefinir a sua password. Clique no link abaixo:</p>
		<p><a href="%s">Redefinir Password</a></p>
		<p>Este link expira em 1 hora.</p>
		<p>Se não solicitou esta alteração, ignore este email.</p>
	`, name, link)

	_ = SendEmail(to, "KPI Platform - Redefinição de Password", body)
}

func EmailWelcome(to, name, tempPassword string) {
	body := fmt.Sprintf(`
		<h2>Bem-vindo à Plataforma KPI</h2>
		<p>Olá %s,</p>
		<p>A sua conta foi criada. Seguem os dados de acesso:</p>
		<ul>
			<li><strong>Email:</strong> %s</li>
			<li><strong>Password temporária:</strong> %s</li>
		</ul>
		<p>Por favor altere a password no primeiro acesso.</p>
	`, name, to, tempPassword)

	_ = SendEmail(to, "KPI Platform - Bem-vindo", body)
}

func EmailTaskUpdated(to, name, taskTitle, updatedBy string) {
	body := fmt.Sprintf(`
		<h2>Tarefa Actualizada</h2>
		<p>Olá %s,</p>
		<p>A tarefa <strong>%s</strong> foi actualizada por %s.</p>
		<p>Aceda à plataforma para mais detalhes.</p>
	`, name, taskTitle, updatedBy)

	_ = SendEmail(to, fmt.Sprintf("KPI Platform - Tarefa Actualizada: %s", taskTitle), body)
}

func EmailBlockerCreated(to, name, blockerDesc, taskTitle string) {
	body := fmt.Sprintf(`
		<h2>Impedimento Reportado</h2>
		<p>Olá %s,</p>
		<p>Um impedimento foi reportado na tarefa <strong>%s</strong>:</p>
		<blockquote>%s</blockquote>
		<p>Aceda à plataforma para aprovar ou rejeitar.</p>
	`, name, taskTitle, blockerDesc)

	_ = SendEmail(to, fmt.Sprintf("KPI Platform - Impedimento: %s", taskTitle), body)
}

func EmailForecastRisk(to, name, taskTitle string, projected, target float64) {
	body := fmt.Sprintf(`
		<h2>Alerta de Previsão</h2>
		<p>Olá %s,</p>
		<p>A tarefa <strong>%s</strong> está em risco de não atingir o objectivo.</p>
		<ul>
			<li><strong>Projecção:</strong> %.0f</li>
			<li><strong>Objectivo:</strong> %.0f</li>
		</ul>
		<p>Aceda à plataforma para tomar acções correctivas.</p>
	`, name, taskTitle, projected, target)

	_ = SendEmail(to, fmt.Sprintf("KPI Platform - ALERTA: %s em risco", taskTitle), body)
}

func EmailMilestoneOverdue(to, name, milestoneTitle, taskTitle string) {
	body := fmt.Sprintf(`
		<h2>Milestone em Atraso</h2>
		<p>Olá %s,</p>
		<p>O milestone <strong>%s</strong> da tarefa <strong>%s</strong> encontra-se em atraso.</p>
		<p>Por favor actualize o progresso na plataforma.</p>
	`, name, milestoneTitle, taskTitle)

	_ = SendEmail(to, fmt.Sprintf("KPI Platform - Atraso: %s", milestoneTitle), body)
}

func EmailBlockerResolved(to, name, taskTitle, status string) {
	body := fmt.Sprintf(`
		<h2>Impedimento Resolvido</h2>
		<p>Olá %s,</p>
		<p>O impedimento na tarefa <strong>%s</strong> foi <strong>%s</strong>.</p>
	`, name, taskTitle, status)

	_ = SendEmail(to, fmt.Sprintf("KPI Platform - Impedimento %s: %s", status, taskTitle), body)
}
