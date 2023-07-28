/*
 * NSSF Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/asaskevich/govalidator"
	"gopkg.in/yaml.v2"

	"github.com/free5gc/nssf/internal/logger"
)

var NssfConfig *Config

func changeSdToLowercase(cfg *Config) {
	// supportedNssaiInPlmnList
	SupportedNssaiInPlmnList := cfg.Configuration.SupportedNssaiInPlmnList
	for i := range SupportedNssaiInPlmnList {
		SupportedSnssaiList := SupportedNssaiInPlmnList[i].SupportedSnssaiList
		for j := range SupportedSnssaiList {
			SupportedSnssaiList[j].Sd = strings.ToLower(SupportedSnssaiList[j].Sd)
		}
	}

	// nsiList
	NsiList := cfg.Configuration.NsiList
	for i := range NsiList {
		NsiList[i].Snssai.Sd = strings.ToLower(NsiList[i].Snssai.Sd)
	}

	// AmfSetList
	AmfSetList := cfg.Configuration.AmfSetList
	for i := range AmfSetList {
		SupportedNssaiAvailabilityData := AmfSetList[i].SupportedNssaiAvailabilityData
		for j := range SupportedNssaiAvailabilityData {
			SupportedSnssaiList := SupportedNssaiAvailabilityData[j].SupportedSnssaiList
			for k := range SupportedSnssaiList {
				SupportedSnssaiList[k].Sd = strings.ToLower(SupportedSnssaiList[k].Sd)
			}
		}
	}

	// AmfList
	AmfList := cfg.Configuration.AmfList
	for i := range AmfList {
		SupportedNssaiAvailabilityData := AmfList[i].SupportedNssaiAvailabilityData
		for j := range SupportedNssaiAvailabilityData {
			SupportedSnssaiList := SupportedNssaiAvailabilityData[j].SupportedSnssaiList
			for k := range SupportedSnssaiList {
				SupportedSnssaiList[k].Sd = strings.ToLower(SupportedSnssaiList[k].Sd)
			}
		}
	}

	// TaList
	TaList := cfg.Configuration.TaList
	for i := range TaList {
		SupportedSnssaiList := TaList[i].SupportedSnssaiList
		for j := range SupportedSnssaiList {
			SupportedSnssaiList[j].Sd = strings.ToLower(SupportedSnssaiList[j].Sd)
		}
	}

	// MappingListFromPlmn
	MappingListFromPlmn := cfg.Configuration.MappingListFromPlmn
	for i := range MappingListFromPlmn {
		MappingOfSnssai := MappingListFromPlmn[i].MappingOfSnssai
		for j := range MappingOfSnssai {
			MappingOfSnssai[j].HomeSnssai.Sd = strings.ToLower(MappingOfSnssai[j].HomeSnssai.Sd)
			MappingOfSnssai[j].ServingSnssai.Sd = strings.ToLower(MappingOfSnssai[j].ServingSnssai.Sd)
		}
	}
}

// TODO: Support configuration update from REST api
func InitConfigFactory(f string, cfg *Config) error {
	if f == "" {
		// Use default config path
		f = NssfDefaultConfigPath
	}

	if content, err := ioutil.ReadFile(f); err != nil {
		return fmt.Errorf("[Factory] %+v", err)
	} else {
		logger.CfgLog.Infof("Read config from [%s]", f)
		if yamlErr := yaml.Unmarshal(content, cfg); yamlErr != nil {
			return fmt.Errorf("[Factory] %+v", yamlErr)
		}
	}
	changeSdToLowercase(cfg)

	return nil
}

func ReadConfig(cfgPath string) (*Config, error) {
	cfg := &Config{}
	if err := InitConfigFactory(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("ReadConfig [%s] Error: %+v", cfgPath, err)
	}
	if _, err := cfg.Validate(); err != nil {
		validErrs := err.(govalidator.Errors).Errors()
		for _, validErr := range validErrs {
			logger.CfgLog.Errorf("%+v", validErr)
		}
		logger.CfgLog.Errorf("[-- PLEASE REFER TO SAMPLE CONFIG FILE COMMENTS --]")
		return nil, fmt.Errorf("Config validate Error")
	}
	return cfg, nil
}
