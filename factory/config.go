/*
 * NSSF Configuration Factory
 */

package factory

import "free5gc/lib/openapi/models"

type Config struct {
	Info *Info `yaml:"info"`

	Configuration *Configuration `yaml:"configuration"`

	Subscriptions []Subscription `yaml:"subscriptions,omitempty"`
}

type Info struct {
	Version string `yaml:"version"`

	Description string `yaml:"description,omitempty"`
}

type Configuration struct {
	NssfName string `yaml:"nssfName,omitempty"`

	Sbi *Sbi `yaml:"sbi"`

	ServiceNameList []models.ServiceName `yaml:"serviceNameList"`

	NrfUri string `yaml:"nrfUri"`

	SupportedPlmnList []models.PlmnId `yaml:"supportedPlmnList,omitempty"`

	SupportedNssaiInPlmnList []SupportedNssaiInPlmn `yaml:"supportedNssaiInPlmnList"`

	NsiList []NsiConfig `yaml:"nsiList,omitempty"`

	AmfSetList []AmfSetConfig `yaml:"amfSetList"`

	AmfList []AmfConfig `yaml:"amfList"`

	TaList []TaConfig `yaml:"taList"`

	MappingListFromPlmn []MappingFromPlmnConfig `yaml:"mappingListFromPlmn"`
}

type Sbi struct {
	Scheme models.UriScheme `yaml:"scheme"`

	// Currently only support IPv4 and thus `Ipv4Addr` field shall not be empty
	RegisterIPv4 string `yaml:"registerIPv4,omitempty"` // IP that is registered at NRF.
	// IPv6Addr string `yaml:"ipv6Addr,omitempty"`
	BindingIPv4 string `yaml:"bindingIPv4,omitempty"` // IP used to run the server in the node.

	Port int `yaml:"port"`
}

type AmfConfig struct {
	NfId string `yaml:"nfId"`

	SupportedNssaiAvailabilityData []models.SupportedNssaiAvailabilityData `yaml:"supportedNssaiAvailabilityData"`
}

type TaConfig struct {
	Tai *models.Tai `yaml:"tai"`

	AccessType *models.AccessType `yaml:"accessType"`

	SupportedSnssaiList []models.Snssai `yaml:"supportedSnssaiList"`

	RestrictedSnssaiList []models.RestrictedSnssai `yaml:"restrictedSnssaiList,omitempty"`
}

type SupportedNssaiInPlmn struct {
	PlmnId *models.PlmnId `yaml:"plmnId"`

	SupportedSnssaiList []models.Snssai `yaml:"supportedSnssaiList"`
}

type NsiConfig struct {
	Snssai *models.Snssai `yaml:"snssai"`

	NsiInformationList []models.NsiInformation `yaml:"nsiInformationList"`
}

type AmfSetConfig struct {
	AmfSetId string `yaml:"amfSetId"`

	AmfList []string `yaml:"amfList,omitempty"`

	NrfAmfSet string `yaml:"nrfAmfSet,omitempty"`

	SupportedNssaiAvailabilityData []models.SupportedNssaiAvailabilityData `yaml:"supportedNssaiAvailabilityData"`
}

type MappingFromPlmnConfig struct {
	OperatorName string `yaml:"operatorName,omitempty"`

	HomePlmnId *models.PlmnId `yaml:"homePlmnId"`

	MappingOfSnssai []models.MappingOfSnssai `yaml:"mappingOfSnssai"`
}

type Subscription struct {
	SubscriptionId string `yaml:"subscriptionId"`

	SubscriptionData *models.NssfEventSubscriptionCreateData `yaml:"subscriptionData"`
}
