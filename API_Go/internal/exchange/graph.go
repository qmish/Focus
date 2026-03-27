package exchange

// Deprecated aliases preserved for backward compatibility.
// Focus now supports only on-prem Exchange/OWA via EWS.
type GraphClient = EWSClient
type GraphConfig = EWSConfig

func NewGraphClient(cfg GraphConfig) (*GraphClient, error) {
	return NewEWSClient(EWSConfig(cfg))
}
