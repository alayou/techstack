package httpd

import (
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/alayou/techstack/global"

	"gopkg.in/gomail.v2"
)

const (
	mailKindAccountActive       = "account-active"
	mailKindPasswordReset       = "password-ResetUserkey"
	mailKindLicenseExpireNotice = "license-expire" //许可到期
)

var (
	mailLinks = map[string]string{
		mailKindAccountActive:       "%s/u/signin/%s",
		mailKindPasswordReset:       "%s/u/password-ResetUserkey/%s",
		mailKindLicenseExpireNotice: "#",
	}
	defaultTemplates = map[string]string{
		mailKindAccountActive:       "<h3>账户激活链接</h3>\n       <p><a href=\"%s\">点击此处账户激活</a></p>\n\t\t<p>如果您没有进行账号注册请忽略！</p>",
		mailKindPasswordReset:       "<h3>密码重置链接</h3>\n       <p><a href=\"%s\">点击此处重置密码</a></p>\n\t\t<p>如果您没有申请重置密码请忽略！</p>",
		mailKindLicenseExpireNotice: "<h3>许可到期提醒</h3>\n       <p><a href=\"%s\">查看</a></p>\n\t\t<p>您的许可快到期，请及时续费！</p>",
	}
	mailOnce sync.Once
	mail     *MailService
)

type MailService struct {
	dialer    *gomail.Dialer
	from      string
	templates map[string]string
	enabled   bool
}

func NewMailService() *MailService {
	mailOnce.Do(func() {
		mail = &MailService{
			templates: defaultTemplates,
		}
		err := mail.Boot()
		if err != nil {
			return
		}
	})
	return mail
}

func (m *MailService) Enabled() bool {
	return m.enabled
}

func (m *MailService) Boot() error {
	configMail := global.Config.Mail
	username := configMail.Username
	password := configMail.Password
	sender := configMail.Sender
	host, port, err := splitHostPort(configMail.Address)
	if err != nil {
		return err
	}

	dialer := gomail.NewDialer(host, port, username, password)
	if _, err = dialer.Dial(); err != nil {
		return err
	}

	m.dialer = dialer
	m.enabled = configMail.Enable
	m.from = fmt.Sprintf("%s <%s>", sender, username)
	return nil
}

func (m *MailService) NotifyActive(siteAddr, email string, token string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "账户激活")
	msg.SetBody("text/html", m.buildMailBody(mailKindAccountActive, siteAddr, email, token))
	return m.dialer.DialAndSend(msg)
}

func (m *MailService) NotifyPasswordReset(siteAddr, email, token string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "密码重置申请")
	msg.SetBody("text/html", m.buildMailBody(mailKindPasswordReset, siteAddr, email, token))
	return m.dialer.DialAndSend(msg)
}

// NotifyLicenseExpire  许可到期
func (m *MailService) NotifyLicenseExpire(email string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.from)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "许可到期提醒")
	msg.SetBody("text/html", m.buildMailBodyText(mailKindLicenseExpireNotice, "#"))
	return m.dialer.DialAndSend(msg)
}

func (m *MailService) buildMailBody(kind, siteAddr, email, token string) string {
	link := fmt.Sprintf(mailLinks[kind], siteAddr, encodeToKey(email, token))
	return fmt.Sprintf(m.templates[kind], link)
}

func (m *MailService) buildMailBodyText(kind, text string) string {
	return fmt.Sprintf(m.templates[kind], text)
}

var base64Encode = base64.URLEncoding.EncodeToString

//var base64Decode = base64.URLEncoding.DecodeString

const zplatSplitKey = "|pass|"

func encodeToKey(email, token string) string {
	return base64Encode([]byte(email + zplatSplitKey + token))
}

func splitHostPort(hostport string) (host string, port int, err error) {
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", 0, fmt.Errorf("invalid smpt-addr: %w", err)
	}

	port, err = strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %w", err)
	}

	return host, port, nil
}
