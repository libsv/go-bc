package bc

// An SPVClient is a struct used to specify interfaces
// used to complete Simple Payment Verification (SPV)
// in conjunction with a Merkle Proof.
type SPVClient struct {
	hcm HeaderChainMapper
}

// NewSPVClient creates a new SPVClient based on params
// passed or will use defaults if nil is passed.
func NewSPVClient(mrr HeaderChainMapper) *SPVClient {
	if mrr == nil {
		return &SPVClient{}
	}

	return &SPVClient{
		hcm: mrr,
	}
}
