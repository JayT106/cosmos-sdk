package capability

import (
	"fmt"
	"os"
	"path"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/capability/keeper"
	"github.com/cosmos/cosmos-sdk/x/capability/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	if err := k.InitializeIndex(ctx, genState.Index); err != nil {
		panic(err)
	}

	// set owners for each index
	for _, genOwner := range genState.Owners {
		k.SetOwners(ctx, genOwner.Index, genOwner.IndexOwners)
	}
	// initialize in-memory capabilities
	k.InitMemStore(ctx)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	index := k.GetLatestIndex(ctx)
	owners := []types.GenesisOwners{}

	for i := uint64(1); i < index; i++ {
		capabilityOwners, ok := k.GetOwners(ctx, i)
		if !ok || len(capabilityOwners.Owners) == 0 {
			continue
		}

		genOwner := types.GenesisOwners{
			Index:       i,
			IndexOwners: capabilityOwners,
		}
		owners = append(owners, genOwner)
	}

	return &types.GenesisState{
		Index:  index,
		Owners: owners,
	}
}

// InitGenesisFrom initializes the capability module's state from a provided genesis
// state.
func InitGenesisFrom(ctx sdk.Context, k keeper.Keeper, importPath string) error {
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
	if err := gs.Unmarshal(bz); err != nil {
		return err
	}

	InitGenesis(ctx, k, gs)
	return nil
}

func ExportGenesisTo(ctx sdk.Context, k keeper.Keeper, exportPath string) error {
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return err
	}

	fp := path.Join(exportPath, fmt.Sprintf("genesis_%s.bin", types.ModuleName))
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	gs := ExportGenesis(ctx, k)
	bz, err := gs.Marshal()
	if err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %s", types.ModuleName, err)
	}

	if _, err := f.Write(bz); err != nil {
		return err
	}

	return nil
}
