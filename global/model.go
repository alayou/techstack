package global

import (
	"net"
	"net/http"
	"net/url"
)

type OS struct {
	Hostname          string     `json:"hostname"`
	GOOS              string     `json:"goos"`
	GOARCH            string     `json:"goarch"`
	GOVERSION         string     `json:"go_version"`
	NumCPU            int        `json:"num_cpu"`
	TmpDir            string     `json:"tmp_dir"`
	HomeDir           string     `json:"home_dir"`
	NetworkInterfaces []net.Addr `json:"network_interfaces"`
}
type RespData struct {
	URL        string      `json:"url"`
	RemoteAddr string      `json:"remote_addr"`
	Args       url.Values  `json:"args"`
	Method     string      `json:"method"`
	Headers    http.Header `json:"headers"`
	Form       url.Values  `json:"form"`
	Data       interface{} `json:"data"`

	JSON  interface{} `json:"json"`
	Files interface{} `json:"files"`

	Route []string
	OS    OS `json:"os"`
}

type Cluster struct {
	Id                   int64  `json:"id,string" gorm:"primaryKey"`
	Name                 string `json:"name"`
	Remark               string `json:"remark"`
	CreatedAt            int64  `json:"created_at"`
	CpuThreshold         int64  `json:"cpu_threshold"`
	MemoryThreshold      int64  `json:"memory_threshold"`
	DiskThreshold        int64  `json:"disk_threshold"`
	DiskIoThreshold      int64  `json:"disk_io_threshold"`
	NetIoThreshold       int64  `json:"net_io_threshold"`
	ColInterval          int64  `json:"col_interval"`
	AlarmRate            int64  `json:"alarm_rate"`
	Disabled             int64  `json:"disabled"`
	NetDownloadThreshold int64  `json:"net_download_threshold"`
}

type Metrics struct {
	CreateTime int64   `yaml:"create_time"`
	Key        string  `yaml:"key"`
	Val        float64 `yaml:"val"`
}

func (r *Metrics) TableName() string {
	return "metrics"
}

// HostMetrics 实时信息
type HostMetrics struct {
	CPUPer      int64 `json:"cpu_per"`
	MemPer      int64 `json:"mem_per"`
	DiskPer     int64 `json:"disk_per"`
	NetWorkDown int64 `json:"net_work_down"`
	NetWorkUp   int64 `json:"net_work_up"`
}

type SessionNotification struct {
	SessionID uint32
	Status    uint32 // 1 在线  2 离线
}

type CPUBaseInfo struct {
	Model   string `json:"model,omitempty"`
	Hz      string `json:"hz,omitempty"`
	CoreNum int32  `json:"core_num,omitempty"`
}

type HostInfo struct {
	Hostname      string `json:"hostname,omitempty"`
	OS            string `json:"os,omitempty"`
	Platform      string `json:"platform,omitempty"`
	KernelVersion string `json:"kernel_version,omitempty"`
	KernelArch    string `json:"kernel_arch,omitempty"`
}

type ConnectOkMsg struct {
	ID int64
	IP string
}

type Strategy struct {
	CpuThreshold    int
	MemoryThreshold int
	DiskThreshold   int
	NetIOByUp       int
	NetIOByDown     int
	ColInterval     int
	AlarmRate       int

	IdleTimeout      int
	TimeoutEventType int
}

type ClientMsg struct {
	Path   string              // from client
	Code   uint32              // from client
	Method string              // from client,to client
	Header map[string][]string // from client,to client
	Body   []byte              // from client,to client
}
