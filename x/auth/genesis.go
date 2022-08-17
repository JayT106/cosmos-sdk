package auth

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
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

func InitGenesisFrom(ctx sdk.Context, cdc codec.JSONCodec, ak keeper.AccountKeeper, importPath string) error {
	f, err := module.OpenGenesisModuleFile(importPath, types.ModuleName)
	if err != nil {
		return err
	}
	defer f.Close()

	bz, err := module.FileRead(f)
	if err != nil {
		return err
	}

	var gs types.GenesisState
	cdc.MustUnmarshalJSON(bz, &gs)
	InitGenesis(ctx, ak, gs)
	return nil
}

// ExportGenesisTo returns a GenesisState for a given context, keeper and export path
func ExportGenesisTo(ctx sdk.Context, cdc codec.JSONCodec, ak keeper.AccountKeeper, exportPath string) error {
	f, err := module.CreateGenesisExportFile(exportPath, types.ModuleName)
	if err != nil {
		return err
	}
	defer f.Close()

	gs := ExportGenesis(ctx, ak)
	bz := cdc.MustMarshalJSON(gs)
	return module.FileWrite(f, bz)
}
