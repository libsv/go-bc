package bc

// An SPVClient is a struct used to specify interfaces
// used to complete Simple Payment Verification (SPV)
// in conjunction with a Merkle Proof.
type SPVClient struct {
	bhc BlockHeaderChain
}

// NewSPVClient creates a new SPVClient based on params
// passed or will use defaults if nil is passed.
func NewSPVClient(bhc BlockHeaderChain) *SPVClient {
	if bhc == nil {
		return &SPVClient{}
	}

	return &SPVClient{
		bhc: bhc,
	}
}
