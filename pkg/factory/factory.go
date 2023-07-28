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
	//supportedNssaiInPlmnList
	for _, SupportedNssaiInPlmn := range cfg.Configuration.SupportedNssaiInPlmnList {
		for _, SupportedSnssai := range SupportedNssaiInPlmn.SupportedSnssaiList {
			SupportedSnssai.Sd = strings.ToLower(SupportedSnssai.Sd)
		}
	}
	//nsiList
	for _, Nsi := range cfg.Configuration.NsiList {
		Nsi.Snssai.Sd = strings.ToLower(Nsi.Snssai.Sd)
	}
	//AmfSetList
	for _, AmfSetConf := range cfg.Configuration.AmfSetList {
		for _, AvailabilityData := range AmfSetConf.SupportedNssaiAvailabilityData {
			for _, Snssai := range AvailabilityData.SupportedSnssaiList {
				Snssai.Sd = strings.ToLower(Snssai.Sd)
			}
		}
	}
	//AmfList
	for _, AmfConf := range cfg.Configuration.AmfList {
		for _, AvailabilityData := range AmfConf.SupportedNssaiAvailabilityData {
			for _, Snssai := range AvailabilityData.SupportedSnssaiList {
				Snssai.Sd = strings.ToLower(Snssai.Sd)
			}
		}
	}
	//TaList
	for _, TaConf := range cfg.Configuration.TaList {
		for _, Snssai := range TaConf.SupportedSnssaiList {
			Snssai.Sd = strings.ToLower(Snssai.Sd)
		}
	}
	//MappingListFromPlmn
	for _, MappingList := range cfg.Configuration.MappingListFromPlmn {
		for _, Mapping := range MappingList.MappingOfSnssai {
			Mapping.HomeSnssai.Sd = strings.ToLower(Mapping.HomeSnssai.Sd)
			Mapping.ServingSnssai.Sd = strings.ToLower(Mapping.ServingSnssai.Sd)
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
