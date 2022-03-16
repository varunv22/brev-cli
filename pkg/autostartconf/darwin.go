package autostartconf

import (
	"errors"
	"fmt"
	"os/exec"
	"path"

	breverrors "github.com/brevdev/brev-cli/pkg/errors"
)

type DarwinServiceType string

const (
	System     DarwinServiceType = "system"
	SingleUser DarwinServiceType = "singleuser"
)

type DarwinPlistConfigurer struct {
	Store           AutoStartStore
	ValueConfigFile string
	ServiceName     string
	ServiceType     DarwinServiceType
}

func (dpc DarwinPlistConfigurer) UnInstall() error {
	plist, err := exec.Command("launchctl", "list", dpc.ServiceName).Output()
	_ = plist // parse it? https://github.com/DHowett/go-plist if we need something.
	running := err == nil
	if running {
		_, err := exec.Command("launchctl", "stop", dpc.ServiceName).CombinedOutput()
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
		_, err = exec.Command("launchctl", "unload", dpc.ServiceName).CombinedOutput()
		if err != nil {
			return breverrors.WrapAndTrace(err)
		}
	}
	destination, err := dpc.GetDestination()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	err = dpc.Store.Remove(destination)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	err = dpc.Store.Remove(targetBin)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	return nil
}

func (dpc DarwinPlistConfigurer) Install() error {
	err := dpc.UnInstall()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	err = dpc.Store.CopyBin(targetBin)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	err = dpc.Store.CopyBin(targetBin)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	destination, err := dpc.GetDestination()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	err = dpc.Store.WriteString(destination, dpc.ValueConfigFile)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	commands, err := dpc.GetExecCommand()
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	err = execCommands(commands)
	if err != nil {
		return breverrors.WrapAndTrace(err)
	}
	return nil
}

func (dpc DarwinPlistConfigurer) GetDestinationDirectory() (string, error) {
	switch dpc.ServiceType {
	case System:
		return "/Library/LaunchDaemons", nil
	case SingleUser:
		homeDir, err := dpc.Store.UserHomeDir()
		if err != nil {
			return "", breverrors.WrapAndTrace(err)
		}
		return path.Join(homeDir, "/Library/LaunchDaemons"), nil

	}
	return "", errors.New("invalid service type")
}

func (dpc DarwinPlistConfigurer) GetDestination() (string, error) {
	destinationDirectory, err := dpc.GetDestinationDirectory()
	if err != nil {
		return "", breverrors.WrapAndTrace(err)
	}
	destination := path.Join(destinationDirectory, dpc.ServiceName+".plist")
	return destination, nil
}

func (dpc DarwinPlistConfigurer) GetExecCommand() ([][]string, error) {
	destination, err := dpc.GetDestination()
	if err != nil {
		return [][]string{}, breverrors.WrapAndTrace(err)
	}
	switch dpc.ServiceType {
	case System:
		return [][]string{
			{"load", "system/" + dpc.ServiceName},
			{"enable", "system/" + dpc.ServiceName},
		}, nil
	case SingleUser:
		return [][]string{{"bootstrap", "gui/" + dpc.Store.GetOSUser(), destination}}, nil

	}
	return [][]string{}, errors.New("invalid service type")
}

func execCommands(commands [][]string) error {
	for _, command := range commands {
		out, err := exec.Command("launchctl", command...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("error running launchctl %s: %v, %s", fmt.Sprint(command), err, out)
		}
	}
	return nil
}
