package auth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/protobuf/proto"
)

// InitGenesis - Init store state from genesis data
//
// CONTRACT: old coins from the FeeCollectionKeeper need to be transferred through
// a genesis port script to the new fee collector account
func InitGenesis(ctx sdk.Context, ak keeper.AccountKeeper, data types.GenesisState) {
	ak.SetParams(ctx, data.Params)

	accounts, err := types.UnpackAccounts(data.Accounts)
	if err != nil {
		panic(err)
	}
	accounts = types.SanitizeGenesisAccounts(accounts)

	for _, a := range accounts {
		acc := ak.NewAccount(ctx, a)
		ak.SetAccount(ctx, acc)
	}

	ak.GetModuleAccount(ctx, types.FeeCollectorName)
}

// ExportGenesis returns a GenesisState for a given context and keeper
func ExportGenesis(ctx sdk.Context, ak keeper.AccountKeeper) *types.GenesisState {
	params := ak.GetParams(ctx)

	var genAccounts types.GenesisAccounts
	ak.IterateAccounts(ctx, func(account types.AccountI) bool {
		genAccount := account.(types.GenesisAccount)
		genAccounts = append(genAccounts, genAccount)
		return false
	})

	return types.NewGenesisState(params, genAccounts)
}

func InitGenesisFrom(ctx sdk.Context, ak keeper.AccountKeeper, data types.GenesisState) error {
	return nil
}

// ExportGenesisTo returns a GenesisState for a given context, keeper and export path
func ExportGenesisTo(ctx sdk.Context, ak keeper.AccountKeeper, exportPath string) error {
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return err
	}

	var fileIndex = 0
	fn := fmt.Sprintf("genesis%d", fileIndex)
	filePath := path.Join(exportPath, fn)
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// write the params
	param := ak.GetParams(ctx)
	encodedParam, err := param.Marshal()
	if err != nil {
		return err
	}

	fs := 0
	offset := 0
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
	offset = fs
	counts := 0
	// leaving space for writing toal account numbers
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, 0)
	n, err = f.Write(b)
	if err != nil {
		return err
	}
	fs += n

	// write the account info into marshal proto message.
	ctxDone := false
	var e = error(nil)

	ak.IterateAccounts(ctx, func(account types.AccountI) bool {
		select {
		case <-ctx.Context().Done():
			ctxDone = true
			return true
		default:
			msg, ok := account.(proto.Message)
			if !ok {
				e = fmt.Errorf("can't protomarshal %T", account)
				return true
			}
			bz, err := proto.Marshal(msg)
			if err != nil {
				e = fmt.Errorf("genesus account marshal err: %s", err)
				return true
			}

			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(len(bz)))
			n, err = f.Write(b)
			if err != nil {
				e = err
				return true
			}
			fs += n

			n, err = f.Write(bz)
			if err != nil {
				e = err
				return true
			}
			fs += n

			// we limited the file size to 100M
			if fs > 100000000 {
				err := f.Close()
				if err != nil {
					e = err
					return true
				}

				fileIndex++
				f, err = os.Create(filePath)
				if err != nil {
					e = err
					return true
				}

				fs = 0
			}

			counts++
			return false
		}
	})

	if ctxDone {
		return errors.New("genesus export terminated")
	}

	if e != nil {
		return e
	}

	// close the current file and reopen the first file and update
	// the account numbers in the file
	err = f.Close()
	if err != nil {
		return err
	}

	fileIndex = 0
	f, err = os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(counts))
	_, err = f.WriteAt(b, int64(offset))
	if err != nil {
		return err
	}

	return nil
}
