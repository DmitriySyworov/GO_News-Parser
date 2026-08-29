package appuseragent

import (
	"github.com/mileusna/useragent"
)

func ValidateUserAgent(oldUserAgent, newUserAgent string) (bool, string) {
	matchUserAgent := true
	oldParse := useragent.Parse(oldUserAgent)
	newParse := useragent.Parse(newUserAgent)
	newDevice := newParse.Device
	newOS := newParse.OS
	newName := newParse.Name
	if oldParse.Device != newDevice ||
		oldParse.Desktop != newParse.Desktop ||
		oldParse.OS != newOS ||
		oldParse.Mobile != newParse.Mobile ||
		oldParse.Tablet != newParse.Tablet ||
		oldParse.Name != newParse.Name {
		matchUserAgent = false
	}
	if newDevice == "" {
		if newOS != "" {
			newDevice = newOS + " (" + newName + ")"
		} else {
			newDevice = newName
		}
	} else {
		newDevice = newName + " (" + newDevice + ")"
	}
	if newDevice == "" {
		newDevice = "unknown"
	}
	return matchUserAgent, newDevice
}
