package slashing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// InitGenesis initialize default parameters
// and the keeper's address to pubkey map
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, stakingKeeper types.StakingKeeper, data *types.GenesisState) {
	stakingKeeper.IterateValidators(ctx,
		func(index int64, validator stakingtypes.ValidatorI) bool {
			consPk, err := validator.ConsPubKey()
			if err != nil {
				panic(err)
			}
			keeper.AddPubkey(ctx, consPk)
			return false
		},
	)

	for _, info := range data.SigningInfos {
		address, err := sdk.ConsAddressFromBech32(info.Address)
		if err != nil {
			panic(err)
		}
		keeper.SetValidatorSigningInfo(ctx, address, info.ValidatorSigningInfo)
	}

	for _, array := range data.MissedBlocks {
		address, err := sdk.ConsAddressFromBech32(array.Address)
		if err != nil {
			panic(err)
		}
		for _, missed := range array.MissedBlocks {
			keeper.SetValidatorMissedBlockBitArray(ctx, address, missed.Index, missed.Missed)
		}
	}

	keeper.SetParams(ctx, data.Params)
}

// ExportGenesis writes the current store values
// to a genesis file, which can be imported again
// with InitGenesis
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) (data *types.GenesisState) {
	params := keeper.GetParams(ctx)
	signingInfos := make([]types.SigningInfo, 0)
	missedBlocks := make([]types.ValidatorMissedBlocks, 0)
	keeper.IterateValidatorSigningInfos(ctx, func(address sdk.ConsAddress, info types.ValidatorSigningInfo) (stop bool) {
		bechAddr := address.String()
		signingInfos = append(signingInfos, types.SigningInfo{
			Address:              bechAddr,
			ValidatorSigningInfo: info,
		})

		localMissedBlocks := keeper.GetValidatorMissedBlocks(ctx, address)

		missedBlocks = append(missedBlocks, types.ValidatorMissedBlocks{
			Address:      bechAddr,
			MissedBlocks: localMissedBlocks,
		})

		return false
	})

	return types.NewGenesisState(params, signingInfos, missedBlocks)
}

func filePath(exportPath string, fileIndex int) string {
	fn := fmt.Sprintf("%s%d", types.ModuleName, fileIndex)
	return path.Join(exportPath, fn)
}

func ExportGenesisTo(ctx sdk.Context, keeper keeper.Keeper, exportPath string) error {
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return err
	}

	var fileIndex = 0
	f, err := os.OpenFile(filePath(exportPath, fileIndex), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	// write the params
	param := keeper.GetParams(ctx)
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
	// leaving space for writing total slashed account numbers
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, 0)
	n, err = f.Write(b)
	if err != nil {
		return err
	}
	fs += n

	// write the slashing info into marshal proto message.
	ctxDone := false
	var e = error(nil)

	keeper.IterateValidatorSigningInfos(ctx, func(address sdk.ConsAddress, info types.ValidatorSigningInfo) (stop bool) {
		select {
		case <-ctx.Context().Done():
			ctxDone = true
			return true
		default:
			bechAddr := address.String()

			si := types.SigningInfo{
				Address:              bechAddr,
				ValidatorSigningInfo: info,
			}
			encoded, err := si.Marshal()
			if err != nil {
				e = fmt.Errorf("failed to marshal")
				return true
			}

			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(len(encoded)))
			n, err := f.Write(b)
			if err != nil {
				e = err
				return true
			}
			fs += n

			n, err = f.Write(encoded)
			if err != nil {
				e = err
				return true
			}
			fs += n

			localMissedBlocks := keeper.GetValidatorMissedBlocks(ctx, address)
			mb := types.ValidatorMissedBlocks{
				Address:      bechAddr,
				MissedBlocks: localMissedBlocks,
			}

			encoded, err = mb.Marshal()
			if err != nil {
				e = fmt.Errorf("failed to marshal")
				return true
			}

			binary.LittleEndian.PutUint32(b, uint32(len(encoded)))
			n, err = f.Write(b)
			if err != nil {
				e = err
				return true
			}
			fs += n

			n, err = f.Write(encoded)
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
				f, err = os.OpenFile(path.Join(exportPath, filePath(exportPath, fileIndex)), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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
	f, err = os.OpenFile(filePath(exportPath, fileIndex), os.O_RDWR, 0666)
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
