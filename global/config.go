package global

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alayou/techstack/model"
	"github.com/gorpher/gone/osutil"
)

type CORS struct {
	AllowOrigin      string `mapstructure:"allow_origin" json:"allow_origin" yaml:"allow_origin"`
	AllowMethods     string `mapstructure:"allow_methods" json:"allow_methods" yaml:"allow_methods"`
	AllowHeaders     string `mapstructure:"allow_headers" json:"allow_headers" yaml:"allow_headers"`
	ExposeHeaders    string `mapstructure:"expose_headers" json:"expose_headers" yaml:"expose_headers"`
	AllowCredentials string `mapstructure:"allow_credentials" json:"allow_credentials" yaml:"allow_credentials"`
}

type Database struct {
	Driver string `json:"driver" yaml:"driver" form:"driver" binding:"required"`
	DSN    string `json:"dsn" yaml:"dsn" form:"dsn" binding:"required"`
}

type Cfg struct {
	ConfigFile string `json:"-" yaml:"-" env:"TECHSTACK_CONFIG"`
	Addr       string `json:"addr" yaml:"addr"`
	Debug      bool   `json:"debug" yaml:"debug"`

	Database Database `json:"database" yaml:"database"`
	LogLevel string   `json:"log_level" yaml:"log_level" env:"LOG_LEVEL"`
	LogFile  string   `json:"log_file" yaml:"log_file" env:"LOG_FILE"`
	Cert     string   `json:"cert" yaml:"cert"`
	Ca       string   `json:"ca" yaml:"ca"`
	Key      string   `json:"key" yaml:"key"`
	CacheDir string   `json:"cache_dir" yaml:"cache_dir"`

	SessionHashKey   string `json:"session_hash_key" yaml:"session_hash_key"`
	SessionCookieKey string `json:"session_cookie_key" yaml:"session_cookie_key"`
	OrderDuration    int64  `json:"order_duration" yaml:"order_duration"`

	// PasswordHashing defines the configuration for password hashing
	PasswordHashing PasswordHashing `json:"password_hashing" mapstructure:"password_hashing"`
	Mail            Mail            `json:"mail" yaml:"mail"` //邮件服务

	Cors CORS `json:"cors" yaml:"cors"`

	LLM model.LLMModelConfig `json:"llm" yaml:"llm"`
}

// Supported algorithms for hashing passwords.
// These algorithms can be used when SFTPGo hashes a plain text password
const (
	HashingAlgoBcrypt   = "bcrypt"
	HashingAlgoArgon2ID = "argon2id"
)

// BcryptOptions defines the options for bcrypt password hashing
type BcryptOptions struct {
	Cost int `json:"cost" mapstructure:"cost"`
}

// Argon2Options defines the options for argon2 password hashing
type Argon2Options struct {
	Memory      uint32 `json:"memory" mapstructure:"memory"`
	Iterations  uint32 `json:"iterations" mapstructure:"iterations"`
	Parallelism uint8  `json:"parallelism" mapstructure:"parallelism"`
}

// PasswordHashing defines the configuration for password hashing
type PasswordHashing struct {
	BcryptOptions BcryptOptions `json:"bcrypt_options" mapstructure:"bcrypt_options"`
	Argon2Options Argon2Options `json:"argon2_options" mapstructure:"argon2_options"`
	// Algorithm to use for hashing passwords. Available algorithms: argon2id, bcrypt. Default: bcrypt
	Algo string `json:"algo" mapstructure:"algo"`
}
type Wxpay struct {
	PrivateKeyPath string `json:"-" yaml:"-"`
	Mchid          string `json:"-" yaml:"-"`
	SerialNo       string `json:"-" yaml:"-"`
	ApiV3Key       string `json:"-" yaml:"-"`
	AppID          string `json:"-" yaml:"-"`
	CallBackURL    string `json:"call_back_url" yaml:"call_back_url"`
}

type Mail struct {
	Enable   bool   `json:"enable" yaml:"enable"`
	Address  string `json:"address" yaml:"address"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Sender   string `json:"sender" yaml:"sender"`
}

var Config = Cfg{
	ConfigFile: "config.yml",
	Addr:       ":36920",
	Debug:      false,
	LogLevel:   "debug",

	SessionHashKey:   "0123456789012345",
	SessionCookieKey: "5432109876543210",
	CacheDir:         ".cache",
	OrderDuration:    60 * 2, // 2 分钟

	PasswordHashing: PasswordHashing{
		Argon2Options: Argon2Options{
			Memory:      65536,
			Iterations:  1,
			Parallelism: 2,
		},
		BcryptOptions: BcryptOptions{
			Cost: 10,
		},
		Algo: HashingAlgoBcrypt,
	},

	Cors: CORS{
		AllowHeaders:     "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token,X-Token,X-User-SourceID",
		AllowMethods:     "POST, GET, OPTIONS,DELETE,PUT",
		AllowOrigin:      "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type",
		AllowCredentials: "true",
	},
}

type License struct {
	HomeURL    string `json:"home_url" yaml:"home_url"`
	PublicKey  string `json:"public_key" yaml:"public_key"`
	PassCert   string `json:"pass_cert" yaml:"pass_cert"`
	BundleCert string `json:"bundle_cert" yaml:"bundle_cert"`
	RootCa     string `json:"root_ca" yaml:"root_ca"`
}

// Sfpd for the SFTP server
type Sfpd struct {
	// Maximum idle timeout as minutes. If a client is idle for a time that exceeds this setting it will be disconnected.
	// 0 means disabled
	IdleTimeout int `json:"idle_timeout" mapstructure:"idle_timeout"`

	// The address to listen on. A blank value means listen on all available network interfaces.
	Address string `json:"address" mapstructure:"address"`
	// The port used for serving requests
	Port int `json:"port" mapstructure:"port"`
	// Apply the proxy configuration, if any, for this binding
	ApplyProxyConfig bool `json:"apply_proxy_config" mapstructure:"apply_proxy_config"`

	// Identification string used by the server
	Banner string `json:"banner" mapstructure:"banner"`

	// Maximum number of authentication attempts permitted per connection.
	// If set to a negative number, the number of attempts is unlimited.
	// If set to zero, the number of attempts are limited to 6.
	MaxAuthTries int `json:"max_auth_tries" mapstructure:"max_auth_tries"`
	// HostKeys define the daemon's private host keys.
	// Each host key can be defined as a path relative to the configuration directory or an absolute one.
	// If empty or missing, the daemon will search or try to generate "id_rsa" and "id_ecdsa" host keys
	// inside the configuration directory.
	HostKeys []string `json:"host_keys" mapstructure:"host_keys"`
	// HostCertificates defines public host certificates.
	// Each certificate can be defined as a path relative to the configuration directory or an absolute one.
	// Certificate's public key must match a private host key otherwise it will be silently ignored.
	HostCertificates []string `json:"host_certificates" mapstructure:"host_certificates"`
	// HostKeyAlgorithms lists the public key algorithms that the server will accept for host
	// key authentication.
	HostKeyAlgorithms []string `json:"host_key_algorithms" mapstructure:"host_key_algorithms"`
	// KexAlgorithms specifies the available KEX (Key Exchange) algorithms in
	// preference order.
	KexAlgorithms []string `json:"kex_algorithms" mapstructure:"kex_algorithms"`
	// Ciphers specifies the ciphers allowed
	Ciphers []string `json:"ciphers" mapstructure:"ciphers"`
	// MACs Specifies the available MAC (message authentication code) algorithms
	// in preference order
	MACs []string `json:"macs" mapstructure:"macs"`
	// PublicKeyAlgorithms lists the supported public key algorithms for client authentication.
	PublicKeyAlgorithms []string `json:"public_key_algorithms" mapstructure:"public_key_algorithms"`
	// TrustedUserCAKeys specifies a list of public keys paths of certificate authorities
	// that are trusted to sign user certificates for authentication.
	// The paths can be absolute or relative to the configuration directory
	TrustedUserCAKeys []string `json:"trusted_user_ca_keys" mapstructure:"trusted_user_ca_keys"`
	// Path to a file containing the revoked user certificates.
	// This file must contain a JSON list with the public key fingerprints of the revoked certificates.
	// Example content:
	// ["SHA256:bsBRHC/xgiqBJdSuvSTNpJNLTISP/G356jNMCRYC5Es","SHA256:119+8cL/HH+NLMawRsJx6CzPF1I3xC+jpM60bQHXGE8"]
	RevokedUserCertsFile string `json:"revoked_user_certs_file" mapstructure:"revoked_user_certs_file"`
	// LoginBannerFile the contents of the specified file, if any, are sent to
	// the remote user before authentication is allowed.
	LoginBannerFile string `json:"login_banner_file" mapstructure:"login_banner_file"`
	// List of enabled SSH commands.
	// We support the following SSH commands:
	// - "scp". SCP is an experimental feature, we have our own SCP implementation since
	//      we can't rely on scp system command to proper handle permissions, quota and
	//      user's home dir restrictions.
	// 		The SCP protocol is quite simple but there is no official docs about it,
	// 		so we need more testing and feedbacks before enabling it by default.
	// 		We may not handle some borderline cases or have sneaky bugs.
	// 		Please do accurate tests yourself before enabling SCP and let us known
	// 		if something does not work as expected for your use cases.
	//      SCP between two remote hosts is supported using the `-3` scp option.
	// - "md5sum", "sha1sum", "sha256sum", "sha384sum", "sha512sum". Useful to check message
	//      digests for uploaded files. These commands are implemented inside SFTPGo so they
	//      work even if the matching system commands are not available, for example on Windows.
	// - "cd", "pwd". Some mobile SFTP clients does not support the SFTP SSH_FXP_REALPATH and so
	//      they use "cd" and "pwd" SSH commands to get the initial directory.
	//      Currently `cd` do nothing and `pwd` always returns the "/" path.
	//
	// The following SSH commands are enabled by default: "md5sum", "sha1sum", "cd", "pwd".
	// "*" enables all supported SSH commands.
	EnabledSSHCommands []string `json:"enabled_ssh_commands" mapstructure:"enabled_ssh_commands"`
	// KeyboardInteractiveAuthentication specifies whether keyboard interactive authentication is allowed.
	// If no keyboard interactive hook or auth plugin is defined the default is to prompt for the user password and then the
	// one time authentication code, if defined.
	KeyboardInteractiveAuthentication bool `json:"keyboard_interactive_authentication" mapstructure:"keyboard_interactive_authentication"`
	// Absolute path to an external program or an HTTP URL to invoke for keyboard interactive authentication.
	// Leave empty to disable this authentication mode.
	KeyboardInteractiveHook string `json:"keyboard_interactive_auth_hook" mapstructure:"keyboard_interactive_auth_hook"`
	// PasswordAuthentication specifies whether password authentication is allowed.
	PasswordAuthentication bool `json:"password_authentication" mapstructure:"password_authentication"`
	// Virtual root folder prefix to include in all file operations (ex: /files).
	// The virtual paths used for per-directory permissions, file patterns etc. must not include the folder prefix.
	// The prefix is only applied to SFTP requests, SCP and other SSH commands will be automatically disabled if
	// you configure a prefix.
	// This setting can help some migrations from OpenSSH. It is not recommended for general usage.
	FolderPrefix string `json:"folder_prefix" mapstructure:"folder_prefix"`
	//certChecker      *ssh.CertChecker
	//parsedUserCAKeys []ssh.PublicKey
}

// GetAddress returns the binding address
func (b *Sfpd) GetAddress() string {
	return fmt.Sprintf("%s:%d", b.Address, b.Port)
}

// IsValid returns true if the binding port is > 0
func (b *Sfpd) IsValid() bool {
	return b.Port > 0
}

func init() {
	location, err := os.Executable()
	if err != nil {
		return
	}
	pwd, _ := os.Getwd() //nolint
	Config.ConfigFile = filepath.Join(pwd, "config.yml")
	if osutil.FileExist(Config.ConfigFile) {
		return
	}
	Config.ConfigFile = filepath.Join(filepath.Dir(location), "config.yml")

}
