package service

import (
	"crowdfunding-backend/config"
	"crowdfunding-backend/internal/repository/interfaces"
	"crowdfunding-backend/internal/util"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
	"gopkg.in/mail.v2"
)

type EmailService struct {
	smtpHost   string
	smtpPort   int
	username   string
	password   string
	userRepo   interfaces.UserRepository
	jwtSecret  string
	domainName string
}

func NewEmailService(userRepo interfaces.UserRepository) *EmailService {
	return &EmailService{
		smtpHost:   config.AppConfig.SMTPHost,
		smtpPort:   config.AppConfig.SMTPPort,
		username:   config.AppConfig.SMTPUsername,
		password:   config.AppConfig.SMTPPassword,
		userRepo:   userRepo,
		jwtSecret:  config.AppConfig.JWTSecret,
		domainName: config.AppConfig.DomainName,
	}
}

func (s *EmailService) SendVerificationEmail(email, username string) error {
	token, err := s.generateEmailVerificationToken(email)
	if err != nil {
		util.Logger.Error("生成验证令牌失败", zap.Error(err))
		return fmt.Errorf("生成验证令牌失败: %w", err)
	}

	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", config.AppConfig.FrontendURL, token)

	subject := "验证您的邮箱"
	body := fmt.Sprintf("亲爱的 %s，\n\n请点击以下链接验证您的邮箱：\n%s\n\n此链接将在24小时后过期。", username, verificationLink)

	s.sendEmailAsync(email, subject, body)
	return nil
}

func (s *EmailService) sendEmailAsync(to, subject, body string) {
	go func() {
		if err := s.sendEmail(to, subject, body); err != nil {
			util.Logger.Error("异步发送邮件失败", zap.Error(err), zap.String("to", to))
		}
	}()
}

func (s *EmailService) sendEmail(to, subject, body string) error {
	util.Logger.Info("SMTP 配置",
		zap.String("SMTPHost", s.smtpHost),
		zap.Int("SMTPPort", s.smtpPort),
		zap.String("SMTPUsername", s.username))

	util.Logger.Info("开始发送邮件",
		zap.String("to", to),
		zap.String("subject", subject))

	m := mail.NewMessage()
	m.SetHeader("From", s.username)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := mail.NewDialer(s.smtpHost, s.smtpPort, s.username, s.password)
	d.Timeout = 20 * time.Second
	d.SSL = true
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// 尝试解析 SMTP 主机
	ips, err := net.LookupIP(s.smtpHost)
	if err != nil {
		util.Logger.Error("无法解析 SMTP 主机", zap.Error(err))
	} else {
		util.Logger.Info("SMTP 主机 IP", zap.Strings("ips", convertIPsToStrings(ips)))
	}

	// 尝试连接 SMTP 服务器
	util.Logger.Info("尝试连接 SMTP 服务器", zap.String("host", s.smtpHost), zap.Int("port", s.smtpPort))
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		util.Logger.Error("无法连接到 SMTP 服务器", zap.Error(err))
		return fmt.Errorf("无法连接到 SMTP 服务器: %w", err)
	}
	conn.Close()

	util.Logger.Info("开始发送邮件")
	if err := d.DialAndSend(m); err != nil {
		util.Logger.Error("发送邮件失败", zap.Error(err))
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	util.Logger.Info("邮件发送成功", zap.String("to", to))
	return nil
}

func convertIPsToStrings(ips []net.IP) []string {
	result := make([]string, len(ips))
	for i, ip := range ips {
		result[i] = ip.String()
	}
	return result
}

func (s *EmailService) generateEmailVerificationToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *EmailService) VerifyEmailToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		util.Logger.Error("解析令牌失败", zap.Error(err))
		return 0, fmt.Errorf("无效的令牌: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		email, ok := claims["email"].(string)
		if !ok {
			util.Logger.Error("令牌中缺少邮箱信息")
			return 0, fmt.Errorf("无效的令牌: 缺少邮箱信息")
		}

		userID, err := s.getUserIDByEmail(email)
		if err != nil {
			util.Logger.Error("获取用户ID失败", zap.Error(err), zap.String("email", email))
			return 0, fmt.Errorf("获取用户ID失败: %w", err)
		}
		return userID, nil
	}

	util.Logger.Error("无效的令牌")
	return 0, fmt.Errorf("无效的令牌")
}

func (s *EmailService) getUserIDByEmail(email string) (int, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return 0, fmt.Errorf("查找用户失败: %w", err)
	}
	if user == nil {
		return 0, fmt.Errorf("未找到用户")
	}
	return user.ID, nil
}

func (s *EmailService) SendPasswordResetEmail(email string) error {
	token, err := s.generatePasswordResetToken(email)
	if err != nil {
		util.Logger.Error("生成密码重置令牌失败", zap.Error(err))
		return fmt.Errorf("生成密码重置令牌失败: %w", err)
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", config.AppConfig.FrontendURL, token)

	subject := "重置您的密码 - JTL Crowd"
	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html lang="zh-CN">
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>重置您的密码</title>
		<style>
			body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; line-height: 1.6; color: #333; background-color: #f4f4f4; }
			.container { max-width: 600px; margin: 0 auto; padding: 20px; background-color: #ffffff; border-radius: 8px; box-shadow: 0 0 10px rgba(0,0,0,0.1); }
			.header { background-color: #1a1a1a; color: #ffffff; padding: 20px; text-align: center; border-top-left-radius: 8px; border-top-right-radius: 8px; }
			.content { padding: 20px; }
			.button { display: inline-block; padding: 12px 24px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 4px; font-weight: bold; }
			.footer { margin-top: 20px; text-align: center; font-size: 0.8em; color: #777; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>JTL Crowd</h1>
			</div>
			<div class="content">
				<h2>密码重置请求</h2>
				<p>亲爱的用户，</p>
				<p>我们收到了您的密码重置请求。如果这不是您本人操作，请忽略此邮件。</p>
				<p>要重置您的密码，请点击下面的按钮：</p>
				<p style="text-align: center;">
					<a href="%s" class="button">重置密码</a>
				</p>
				<p>或者，您可以将以下链接复制并粘贴到您的浏览器地址栏：</p>
				<p>%s</p>
				<p>此链接将在1小时后过期。</p>
				<p>如果您没有请求重置密码，请忽略此邮件，您的账户将保持安全。</p>
			</div>
			<div class="footer">
				<p>此邮件由系统自动发送，请勿直接回复。</p>
				<p>&copy; 2024 JTL Crowd. 保留所有权利。</p>
			</div>
		</div>
	</body>
	</html>
	`, resetLink, resetLink)

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) generatePasswordResetToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"type":  "password_reset",
	})
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *EmailService) VerifyPasswordResetToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		util.Logger.Error("解析密码重置令牌失败", zap.Error(err))
		return "", fmt.Errorf("无效的令牌: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		email, ok := claims["email"].(string)
		if !ok {
			util.Logger.Error("令牌中缺少邮箱信息")
			return "", fmt.Errorf("无效的令牌: 缺少邮箱信息")
		}

		tokenType, ok := claims["type"].(string)
		if !ok || tokenType != "password_reset" {
			util.Logger.Error("无效的令牌类型")
			return "", fmt.Errorf("无效的令牌类型")
		}

		return email, nil
	}

	util.Logger.Error("无效的令牌")
	return "", fmt.Errorf("无效的令牌")
}
