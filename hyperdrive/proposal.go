package hyper

import (
	"context"

	"github.com/republicprotocol/republic-go/identity"
)

type Signer interface {
	Sign() Signature
	ID() identity.ID
}

type Signature string
type Block struct {
	Tuples
	Signature
}
type Proposal struct {
	Signature
	Block
	Rank
	Height uint64
}

func ProcessProposal(ctx context.Context, proposalChIn <-chan Proposal, validator Validator) (<-chan Prepare, <-chan Fault, <-chan error) {
	prepareCh := make(chan Prepare, validator.Threshold())
	faultCh := make(chan Fault, validator.Threshold())
	errCh := make(chan error, validator.Threshold())
	counter := uint64(0)

	go func() {
		defer close(prepareCh)
		defer close(faultCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case proposal, ok := <-proposalChIn:
				counter++
				if !ok {
					return
				}
				if validator.ValidateProposal(proposal) {
					prepare := Prepare{
						validator.Sign(),
						proposal.Block,
						proposal.Rank,
						proposal.Height,
					}
					// log.Println("Validated proposal on", validator.Sign())
					prepareCh <- prepare
				} else {
					fault := Fault{
						proposal.Rank,
						proposal.Height,
						validator.Sign(),
					}
					select {
					case <-ctx.Done():
						errCh <- ctx.Err()
						return
					case faultCh <- fault:
					}
				}
			}
		}
	}()

	return prepareCh, faultCh, errCh
}

// func signProposal(p Proposal, signer Signer) (Proposal, error) {
// 	b, err := signBlock(p.Block, signer)
// 	if err != nil {
// 		return Proposal{}, err
// 	}
// 	p.Block = b
// 	var proposalBuf bytes.Buffer
// 	binary.Write(&proposalBuf, binary.BigEndian, p)
// 	sig, err := signer.Sign(proposalBuf.Bytes())
// 	return Proposal{
// 		sig,
// 		p.Block,
// 		p.Rank,
// 		p.Height,
// 	}, nil
// }

// func signBlock(b Block, signer Signer) (Block, error) {
// 	var blockBuf bytes.Buffer
// 	binary.Write(&blockBuf, binary.BigEndian, b)
// 	sig, err := signer.Sign(blockBuf.Bytes())
// 	if err != nil {
// 		return Block{}, err
// 	}
// 	return Block{
// 		b.tuples,
// 		sig,
// 	}, nil
// }