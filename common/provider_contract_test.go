package pan

import "testing"

// TestProviderContractRegistry verifies the no-network contract shared by all
// provider implementations: the factory can construct each registered service,
// the service reports the requested type, and the common PanService interface
// is available. Provider-specific HTTP behavior remains covered by its own
// parser/login tests and is never exercised with real credentials here.
func TestProviderContractRegistry(t *testing.T) {
	cases := []struct {
		name     string
		provider ServiceType
		url      string
	}{
		{"quark", Quark, "https://pan.quark.cn/s/contract-test"},
		{"alipan", Alipan, "https://www.alipan.com/s/contract-test"},
		{"baidu", BaiduPan, "https://pan.baidu.com/s/contract-test"},
		{"uc", UC, "https://drive.uc.cn/s/contract-test"},
		{"xunlei", Xunlei, "https://pan.xunlei.com/s/contract-test"},
	}
	factory := NewPanFactory()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractServiceType(tc.url); got != tc.provider {
				t.Fatalf("ExtractServiceType() = %v, want %v", got, tc.provider)
			}
			service, err := factory.CreatePanServiceByType(tc.provider, &PanConfig{URL: tc.url, Cookie: "test-only"})
			if err != nil {
				t.Fatalf("CreatePanServiceByType() error: %v", err)
			}
			if service == nil {
				t.Fatal("CreatePanServiceByType() returned nil service")
			}
			if service.GetServiceType() != tc.provider {
				t.Fatalf("GetServiceType() = %v, want %v", service.GetServiceType(), tc.provider)
			}
			var _ PanService = service
		})
	}
}

func TestProviderContractRegistryRejectsUnimplementedProviders(t *testing.T) {
	for _, provider := range []ServiceType{Tianyi, Pan123, Pan115, NotFound} {
		if service, err := NewPanFactory().CreatePanServiceByType(provider, &PanConfig{}); err == nil || service != nil {
			t.Fatalf("provider %s should remain unavailable, service=%v err=%v", provider, service, err)
		}
	}
}
