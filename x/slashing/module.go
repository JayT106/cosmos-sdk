package slashing

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/slashing/client/cli"
	"github.com/cosmos/cosmos-sdk/x/slashing/client/rest"
	"github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/cosmos/cosmos-sdk/x/slashing/simulation"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

// AppModuleBasic defines the basic application module used by the slashing module.
type AppModuleBasic struct {
	cdc codec.Codec
}

var _ module.AppModuleBasic = AppModuleBasic{}

// Name returns the slashing module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the slashing module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the module's interface types
func (b AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the slashing
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the slashing module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return types.ValidateGenesis(data)
}

// RegisterRESTRoutes registers the REST routes for the slashing module.
func (AppModuleBasic) RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {
	rest.RegisterHandlers(clientCtx, rtr)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the slashig module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
}

// GetTxCmd returns the root tx command for the slashing module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetQueryCmd returns no root query command for the slashing module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule implements an application module for the slashing module.
type AppModule struct {
	AppModuleBasic

	keeper        keeper.Keeper
	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper, sk types.StakingKeeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         keeper,
		accountKeeper:  ak,
		bankKeeper:     bk,
		stakingKeeper:  sk,
	}
}

// Name returns the slashing module's name.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterInvariants registers the slashing module invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// Route returns the message routing key for the slashing module.
func (am AppModule) Route() sdk.Route {
	return sdk.NewRoute(types.RouterKey, NewHandler(am.keeper))
}

// QuerierRoute returns the slashing module's querier route name.
func (AppModule) QuerierRoute() string {
	return types.QuerierRoute
}

// LegacyQuerierHandler returns the slashing module sdk.Querier.
func (am AppModule) LegacyQuerierHandler(legacyQuerierCdc *codec.LegacyAmino) sdk.Querier {
	return keeper.NewQuerier(am.keeper, legacyQuerierCdc)
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	m := keeper.NewMigrator(am.keeper)
	cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2)
}

// InitGenesis performs genesis initialization for the slashing module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.keeper, am.stakingKeeper, &genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the slashing
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 2 }

// BeginBlock returns the begin blocker for the slashing module.
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	BeginBlocker(ctx, req, am.keeper)
}

// EndBlock returns the end blocker for the slashing module. It returns no validator
// updates.
func (AppModule) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the slashing module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(simState module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized slashing param changes for the simulator.
func (AppModule) RandomizedParams(r *rand.Rand) []simtypes.ParamChange {
	return simulation.ParamChanges(r)
}

// RegisterStoreDecoder registers a decoder for slashing module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simulation.NewDecodeStore(am.cdc)
}

// WeightedOperations returns the all the slashing module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return simulation.WeightedOperations(
		simState.AppParams, simState.Cdc,
		am.accountKeeper, am.bankKeeper, am.keeper, am.stakingKeeper,
	)
}

// InitGenesisFrom performs genesis initialization for the slashing module. It returns
// no validator updates.
func (am AppModule) InitGenesisFrom(ctx sdk.Context, cdc codec.JSONCodec, path string) ([]abci.ValidatorUpdate, error) {
	// var genesisState types.GenesisState
	// cdc.MustUnmarshalJSON(data, &genesisState)
	// InitGenesis(ctx, am.keeper, am.stakingKeeper, &genesisState)
	return []abci.ValidatorUpdate{}, nil
}

// ExportGenesisTo exports the genesis state as raw bytes files to the destination
// path for the slashing module.
func (am AppModule) ExportGenesisTo(ctx sdk.Context, cdc codec.JSONCodec, exportPath string) error {
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
	param := am.keeper.GetParams(ctx)
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

	am.keeper.IterateValidatorSigningInfos(ctx, func(address sdk.ConsAddress, info types.ValidatorSigningInfo) (stop bool) {
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

			localMissedBlocks := am.keeper.GetValidatorMissedBlocks(ctx, address)
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
