package keeper

import (
	"fmt"
	"os"
	"path"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/crisis/types"
)

// new crisis genesis
func (k Keeper) InitGenesis(ctx sdk.Context, data *types.GenesisState) {
	k.SetConstantFee(ctx, data.ConstantFee)
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	constantFee := k.GetConstantFee(ctx)
	return types.NewGenesisState(constantFee)
}

func (k Keeper) InitGenesisFrom(ctx sdk.Context, cdc codec.JSONCodec, importPath string) error {
	fp := path.Join(importPath, fmt.Sprintf("genesis_%s.bin", types.ModuleName))
	f, err := os.OpenFile(fp, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	bz := make([]byte, fi.Size())
	if _, err := f.Read(bz); err != nil {
		return err
	}

	var gs types.GenesisState
	cdc.MustUnmarshalJSON(bz, &gs)
	k.SetConstantFee(ctx, gs.ConstantFee)
	return nil
}

func (k Keeper) ExportGenesisTo(ctx sdk.Context, cdc codec.JSONCodec, exportPath string) error {
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return err
	}

	fp := path.Join(exportPath, fmt.Sprintf("genesis_%s.bin", types.ModuleName))
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	gs := k.ExportGenesis(ctx)
	bz := cdc.MustMarshalJSON(gs)
	if _, err = f.Write(bz); err != nil {
		return err
	}

	return nil
}
