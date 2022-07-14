package keeper

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

// InitGenesis initializes the bank module's state from a given genesis state.
func (k BaseKeeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	totalSupply := sdk.Coins{}

	genState.Balances = types.SanitizeGenesisBalances(genState.Balances)
	for _, balance := range genState.Balances {
		addr := balance.GetAddress()

		if err := k.initBalances(ctx, addr, balance.Coins); err != nil {
			panic(fmt.Errorf("error on setting balances %w", err))
		}

		totalSupply = totalSupply.Add(balance.Coins...)
	}

	if !genState.Supply.Empty() && !genState.Supply.IsEqual(totalSupply) {
		panic(fmt.Errorf("genesis supply is incorrect, expected %v, got %v", genState.Supply, totalSupply))
	}

	for _, supply := range totalSupply {
		k.setSupply(ctx, supply)
	}

	for _, meta := range genState.DenomMetadata {
		k.SetDenomMetaData(ctx, meta)
	}
}

// ExportGenesis returns the bank module's genesis state.
func (k BaseKeeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	totalSupply, _, err := k.GetPaginatedTotalSupply(ctx, &query.PageRequest{Limit: query.MaxLimit})
	if err != nil {
		panic(fmt.Errorf("unable to fetch total supply %v", err))
	}

	return types.NewGenesisState(
		k.GetParams(ctx),
		k.GetAccountsBalances(ctx),
		totalSupply,
		k.GetAllDenomMetaData(ctx),
	)
}

func (k BaseKeeper) ExportGenesisTo(ctx sdk.Context, exportPath string) error {
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return err
	}

	var fileIndex = 0
	fn := fmt.Sprintf("%s%d", types.ModuleName, fileIndex)
	filePath := path.Join(exportPath, fn)
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	// write the params
	param := k.GetParams(ctx)
	encodedParam, err := param.Marshal()
	if err != nil {
		return err
	}

	fs := 0
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(len(encodedParam)))
	n, err := f.Write(b)
	if err != nil {
		return err
	}
	fs += n

	n, err = f.Write(encodedParam)
	if err != nil {
		return err
	}
	fs += n

	balances := k.GetAccountsBalances(ctx)
	if balances == nil {
		return fmt.Errorf("genesis export context is closed")
	}

	// write the total account numbers
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(balances)))
	n, err = f.Write(b)
	if err != nil {
		return err
	}
	fs += n

	// write the account balances
	for _, balance := range balances {
		select {
		case <-ctx.Context().Done():
			return fmt.Errorf("genesis export context is closed")
		default:
			bz, err := balance.Marshal()
			if err != nil {
				return err
			}

			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(len(bz)))
			n, err := f.Write(b)
			if err != nil {
				return err
			}
			fs += n

			n, err = f.Write(bz)
			if err != nil {
				return err
			}
			fs += n

			if fs > 100000000 {
				err := f.Close()
				if err != nil {
					return err
				}

				fileIndex++
				f, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				if err != nil {
					return err
				}
				fs = 0
			}
		}
	}

	coinsSupply, _, err := k.GetPaginatedTotalSupply(ctx, &query.PageRequest{Limit: query.MaxLimit})
	if err != nil {
		return fmt.Errorf("unable to fetch total supply %v", err)
	}

	// write the total coin numbers
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(coinsSupply)))
	n, err = f.Write(b)
	if err != nil {
		return err
	}
	fs += n

	// write the coin supply
	for _, coinSupply := range coinsSupply {
		select {
		case <-ctx.Context().Done():
			return fmt.Errorf("genesis export context is closed")
		default:
			bz, err := coinSupply.Marshal()
			if err != nil {
				return err
			}

			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(len(bz)))
			n, err := f.Write(b)
			if err != nil {
				return err
			}
			fs += n

			n, err = f.Write(bz)
			if err != nil {
				return err
			}
			fs += n

			if fs > 100000000 {
				err := f.Close()
				if err != nil {
					return err
				}

				fileIndex++
				f, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				if err != nil {
					return err
				}
				fs = 0
			}
		}
	}

	// write the denominations metadata numbers
	mds := k.GetAllDenomMetaData(ctx)
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(mds)))
	n, err = f.Write(b)
	if err != nil {
		return err
	}
	fs += n

	for _, md := range mds {
		select {
		case <-ctx.Context().Done():
			return fmt.Errorf("genesis export context is closed")
		default:
			bz, err := md.Marshal()
			if err != nil {
				return err
			}

			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(len(bz)))
			n, err := f.Write(b)
			if err != nil {
				return err
			}
			fs += n

			n, err = f.Write(bz)
			if err != nil {
				return err
			}
			fs += n

			if fs > 100000000 {
				err := f.Close()
				if err != nil {
					return err
				}

				fileIndex++
				f, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				if err != nil {
					return err
				}
				fs = 0
			}
		}
	}

	return nil
}
