package server

// DONTCOVER

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/db/rocksdb"
	pruningtypes "github.com/cosmos/cosmos-sdk/pruning/types"
	"github.com/cosmos/cosmos-sdk/store/rootmulti"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/store/v2alpha1/multi"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	dbm "github.com/tendermint/tm-db"
)

// ExportCmd dumps app state to JSON.
func DBMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dbmigrate",
		Short: "store migrate from v1 to v2",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := GetServerContextFromCmd(cmd)

			home, _ := cmd.Flags().GetString(flags.FlagHome)
			dst, _ := cmd.Flags().GetString(flags.FlagDst)

			fmt.Printf("migrating home: %s, dst: %s\n", home, dst)

			db, err := openDB(home, dbm.RocksDBBackend)
			if err != nil {
				return err
			}

			defer func() {
				err = db.Close()
				if err != nil {
					fmt.Printf("error closing db: %s\n", err)
				}
			}()

			evmKey := sdktypes.NewKVStoreKey("evm")
			v1Store := rootmulti.NewStore(db, ctx.Logger)
			v1Store.MountStoreWithDB(evmKey, storetypes.StoreTypeIAVL, nil)
			err = v1Store.LoadLatestVersion()
			if err != nil {
				return err
			}

			dbSS, err := rocksdb.NewDB(dst + "_ss")
			if err != nil {
				return err
			}

			defer func() {
				err = dbSS.Close()
				if err != nil {
					fmt.Printf("error closing dbSS: %s\n", err)
				}
			}()

			dbSC, err := rocksdb.NewDB(dst + "_sc")
			if err != nil {
				return err
			}

			defer func() {
				err = dbSC.Close()
				if err != nil {
					fmt.Printf("error closing dbSC: %s\n", err)
				}
			}()

			storeConfig := multi.DefaultStoreConfig()
			storeConfig.Pruning = pruningtypes.NewPruningOptions(pruningtypes.PruningNothing)
			storeConfig.StateCommitmentDB = dbSC

			start := time.Now()
			v2Store, err := multi.MigrateFromV1(v1Store, dbSS, storeConfig)
			timespend := time.Now().Sub(start)
			fmt.Printf("migration took %s\n", timespend)

			id := v2Store.LastCommitID()
			fmt.Printf("last commit id: %s\n", id)

			if err != nil {
				return err
			}

			defer func() {
				err = v2Store.Close()
				if err != nil {
					fmt.Printf("error closing v2Store: %s\n", err)
				}
			}()

			scIt, err := v2Store.StateCommitmentDB.Reader().Iterator(nil, nil)
			if err != nil {
				return err
			}

			defer func() {
				err = scIt.Close()
				if err != nil {
					fmt.Printf("error closing scIt: %s\n", err)
				}
			}()

			scSize := 0
			scCount := 0
			fmt.Printf("iterating over sc\n")
			for scIt.Next() {
				scSize += len(scIt.Key()) + len(scIt.Value())
				scCount++
			}

			fmt.Printf("sc size: %d, count: %d\n", scSize, scCount)

			ssIt, err := v2Store.GetStoreReader().Iterator(nil, nil)
			if err != nil {
				return err
			}

			defer func() {
				err = ssIt.Close()
				if err != nil {
					fmt.Printf("error closing ssIt: %s\n", err)
				}
			}()

			ssSize := 0
			ssCount := 0
			fmt.Printf("iterating over ss\n")
			for ssIt.Next() {
				ssSize += len(ssIt.Key()) + len(ssIt.Value())
				ssCount++
			}

			fmt.Printf("ss size: %d, count: %d\n", ssSize, ssCount)
			fmt.Printf("total v2 size: %d, count: %d\n", scSize+ssSize, scCount+ssCount)

			fmt.Printf("migrated from v1 to v2\n")
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, "home", "The application home directory")
	cmd.Flags().String(flags.FlagDst, "dst", "The migrating db dst")

	return cmd
}
