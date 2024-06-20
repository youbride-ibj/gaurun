package gaurun

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync/atomic"

	"github.com/pelletier/go-toml"
)

type ConfToml struct {
	Core    SectionCore    `toml:"core"`
	Android SectionAndroid `toml:"android"`
	Ios     SectionIos     `toml:"ios"`
	Log     SectionLog     `toml:"log"`
}

type SectionCore struct {
	Port               string `toml:"port"`
	WorkerNum          int64  `toml:"workers"`
	QueueNum           int64  `toml:"queues"`
	NotificationMax    int64  `toml:"notification_max"`
	PusherMax          int64  `toml:"pusher_max"`
	ShutdownTimeout    int64  `toml:"shutdown_timeout"`
	Pid                string `toml:"pid"`
	AllowsEmptyMessage bool   `toml:"allows_empty_message"`
}

type SectionAndroid struct {
	Enabled               bool   `toml:"enabled"`
	ApiKey                string `toml:"apikey"`
	Timeout               int    `toml:"timeout"`
	KeepAliveTimeout      int    `toml:"keepalive_timeout"`
	KeepAliveConns        int    `toml:"keepalive_conns"`
	RetryMax              int    `toml:"retry_max"`
	ServiceAccountKeyPath string `toml:"service_account_key_path"`
	ProjectId             string `toml:"project_id"`
}

type SectionIos struct {
	Enabled          bool   `toml:"enabled"`
	PemCertPath      string `toml:"pem_cert_path"`
	PemKeyPath       string `toml:"pem_key_path"`
	PemKeyPassphrase string `toml:"pem_key_passphrase"`
	TokenAuthKeyPath string `toml:"token_auth_key_path"`
	TokenAuthKeyID   string `toml:"token_auth_key_id"`
	TokenAuthTeamID  string `toml:"token_auth_team_id"`
	Sandbox          bool   `toml:"sandbox"`
	RetryMax         int    `toml:"retry_max"`
	Timeout          int    `toml:"timeout"`
	KeepAliveTimeout int    `toml:"keepalive_timeout"`
	KeepAliveConns   int    `toml:"keepalive_conns"`
	Topic            string `toml:"topic"`
}

type SectionLog struct {
	AccessLog string `toml:"access_log"`
	ErrorLog  string `toml:"error_log"`
	Level     string `toml:"level"`
}

func BuildDefaultConf() ConfToml {
	numCPU := runtime.NumCPU()

	var conf ConfToml
	// Core
	conf.Core.Port = "1056"
	conf.Core.WorkerNum = int64(numCPU)
	conf.Core.QueueNum = 8192
	conf.Core.NotificationMax = 100
	conf.Core.PusherMax = 0
	conf.Core.ShutdownTimeout = 10
	conf.Core.Pid = ""
	conf.Core.AllowsEmptyMessage = false
	// Android
	conf.Android.ApiKey = ""
	conf.Android.Enabled = true
	conf.Android.Timeout = 5
	conf.Android.KeepAliveTimeout = 90
	conf.Android.KeepAliveConns = numCPU
	conf.Android.RetryMax = 1
	conf.Android.ServiceAccountKeyPath = ""
	conf.Android.ProjectId = ""
	// iOS
	conf.Ios.Enabled = true
	conf.Ios.PemCertPath = ""
	conf.Ios.PemKeyPath = ""
	conf.Ios.TokenAuthKeyPath = ""
	conf.Ios.TokenAuthKeyID = ""
	conf.Ios.TokenAuthTeamID = ""
	conf.Ios.Sandbox = true
	conf.Ios.RetryMax = 1
	conf.Ios.Timeout = 5
	conf.Ios.KeepAliveTimeout = 90
	conf.Ios.KeepAliveConns = numCPU
	conf.Ios.Topic = ""
	// log
	conf.Log.AccessLog = "stdout"
	conf.Log.ErrorLog = "stderr"
	conf.Log.Level = "error"
	return conf
}

func LoadConf(confGaurun ConfToml, confPath string) (ConfToml, error) {
	doc, err := ioutil.ReadFile(confPath)
	if err != nil {
		return confGaurun, err
	}
	err = toml.Unmarshal(doc, &confGaurun)
	if err != nil {
		return confGaurun, err
	}
	return confGaurun, nil
}

func ConfigPushersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		sendResponse(w, "method must be PUT", http.StatusBadRequest)
		return
	}

	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		LogError.Error(err.Error())
		sendResponse(w, "url parameters could not be parsed", http.StatusBadRequest)
		return
	}

	in := ""
	for k, v := range values {
		if k == "max" {
			in = v[0]
			break
		}
	}

	if in == "" {
		sendResponse(w, "malformed value", http.StatusBadRequest)
		return
	}

	newPusherMax, err := strconv.ParseInt(in, 0, 64)
	if err != nil {
		LogError.Error(err.Error())
		sendResponse(w, "malformed value", http.StatusBadRequest)
		return
	}

	if newPusherMax < 0 {
		sendResponse(w, "malformed value", http.StatusBadRequest)
		return
	}

	atomic.StoreInt64(&ConfGaurun.Core.PusherMax, newPusherMax)

	sendResponse(w, "ok", http.StatusOK)
}

func (s *SectionIos) IsTokenBasedProvider() bool {
	return s.TokenAuthKeyPath != "" && s.TokenAuthKeyID != "" && s.TokenAuthTeamID != ""
}

func (s *SectionIos) IsCertificateBasedProvider() bool {
	return s.PemCertPath != "" && s.PemKeyPath != ""
}
