//go:build windows

package collector

import "golang.org/x/sys/windows"

func isAdministrator() bool {
	sid, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return false
	}

	member, err := windows.Token(0).IsMember(sid)
	return err == nil && member
}
