package keeper

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
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

func (k Keeper) InitGenesisFrom(ctx sdk.Context, importPath string) error {
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

	var gs *types.GenesisState
	start := time.Now()
	if err := gs.Unmarshal(bz); err != nil {
		return err
	}
	telemetry.MeasureSince(start, "InitGenesis", "crisis", "unmarshal")

	k.SetConstantFee(ctx, gs.ConstantFee)
	return nil
}

func (k Keeper) ExportGenesisTo(ctx sdk.Context, exportPath string) error {
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
	bz, err := gs.Marshal()
	if err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %s", types.ModuleName, err)
	}

	if _, err = f.Write(bz); err != nil {
		return err
	}

	return nil
}
