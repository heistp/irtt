// +build !windows,!nacl,!plan9

package irtt

import (
	"log/syslog"
	"net/url"
)

const syslogSupport = true

const defaultSyslogTag = "irtt"

type syslogHandler struct {
	syslogWriter *syslog.Writer
}

func (s *syslogHandler) OnEvent(e *Event) {
	if s.syslogWriter != nil {
		if e.IsError() {
			s.syslogWriter.Err(e.String())
		} else {
			s.syslogWriter.Info(e.String())
		}
	}
}

func newSyslogHandler(uriStr string) (sh Handler, err error) {
	var suri *url.URL
	if suri, err = parseSyslogURI(uriStr); err != nil {
		return
	}

	prio := syslog.LOG_DAEMON | syslog.LOG_INFO
	var sw *syslog.Writer
	if suri.Scheme == "local" {
		sw, err = syslog.New(prio, suri.Path)
	} else {
		sw, err = syslog.Dial(suri.Scheme, suri.Host, prio, suri.Path)
	}
	sh = &syslogHandler{sw}

	return
}

func parseSyslogURI(suri string) (u *url.URL, err error) {
	if u, err = url.Parse(suri); err != nil {
		return
	}
	if u.Path[0] == '/' {
		u.Path = u.Path[1:]
	}
	if u.Path == "" {
		u.Path = defaultSyslogTag
	}
	return
}
