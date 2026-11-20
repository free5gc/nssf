package context

import (
	"testing"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/serviceuri"
)

// The registered NFService must follow TS 29.510 (no URL in apiPrefix, a
// hostname as fqdn, an IP literal in ipEndPoints) and resolve back to the NSSF
// API root.
func TestNFServiceAddressing(t *testing.T) {
	tests := []struct {
		registerAddr string
		wantFQDN     string
		wantIPv4     string
		wantAPIRoot  string
	}{
		{"10.0.0.6", "", "10.0.0.6", "http://10.0.0.6:8000"},
		{"nssf.free5gc.org", "nssf.free5gc.org", "", "http://nssf.free5gc.org:8000"},
	}

	for _, tt := range tests {
		t.Run(tt.registerAddr, func(t *testing.T) {
			nssfContext.UriScheme = models.UriScheme_HTTP
			nssfContext.RegisterIPv4 = tt.registerAddr
			nssfContext.SBIPort = 8000
			name := models.Nrf_NFMgmt_ServiceName_NNSSF_NSSELECTION
			s := initNfService([]models.Nrf_NFMgmt_ServiceName{name}, "1.0.0")[name]

			if s.ApiPrefix != "" || s.Fqdn != tt.wantFQDN ||
				len(s.IpEndPoints) != 1 || s.IpEndPoints[0].Ipv4Address != tt.wantIPv4 {
				t.Fatalf("unexpected NFService addressing: %+v", s)
			}

			profile := &models.Nrf_NFDisc_NFProfile{NfServices: []models.Nrf_NFDisc_NFService{{
				ServiceName:     s.ServiceName,
				NfServiceStatus: s.NfServiceStatus,
				Scheme:          s.Scheme,
				Fqdn:            s.Fqdn,
				IpEndPoints:     s.IpEndPoints,
			}}}
			got, err := serviceuri.ResolveNFServiceURI(profile, s.ServiceName, serviceuri.Options{})
			if err != nil || got != tt.wantAPIRoot {
				t.Fatalf("resolved %q, error %v; want %q", got, err, tt.wantAPIRoot)
			}
		})
	}
}
