package server

// DONTCOVER

import (
	"crypto/rand"
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

// StoreCmpCmd .
func StoreCmpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storecompare",
		Short: "store compare between v1 and v2",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := GetServerContextFromCmd(cmd)

			dst, _ := cmd.Flags().GetString(flags.FlagDst)

			fmt.Printf("storecompare dst: %s\n", dst)

			db, err := openDB(dst, dbm.RocksDBBackend)
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
			err = v1Store.LoadVersion(0)
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
			if err = storeConfig.RegisterSubstore(evmKey.Name(), storetypes.StoreTypePersistent); err != nil {
				return err
			}

			v2Store, err := multi.NewStore(dbSS, storeConfig)
			if err != nil {
				return err
			}
			defer func() {
				err = v2Store.Close()
				if err != nil {
					fmt.Printf("error closing dbSC: %s\n", err)
				}
			}()

			lastIndex := 0
			v1Time := int64(0)
			v2Time := int64(0)
			repeat := false
			for v := 0; v < 1000; v++ {
				v1KVStore := v1Store.GetCommitKVStore(evmKey)
				v2KVStore := v2Store.GetKVStore(evmKey)

				kvPerCommit := 100
				kvRepeatPerCommit := 10

				if repeat {
					for i := lastIndex - 1; i >= lastIndex-kvRepeatPerCommit; i-- {
						key := fmt.Sprintf("%053d", lastIndex)
						value := make([]byte, 32)
						_, _ = rand.Read(value)
						v1KVStore.Set([]byte(key), value)
						v2KVStore.Set([]byte(key), value)
					}
				}

				for i := lastIndex; i < (v+1)*kvPerCommit; i++ {
					key := fmt.Sprintf("%053d", i)
					value := make([]byte, 32)
					_, _ = rand.Read(value)
					v1KVStore.Set([]byte(key), value)
					v2KVStore.Set([]byte(key), value)
					lastIndex++
				}

				start := time.Now()
				id := v1Store.Commit()
				v1Time += time.Since(start).Nanoseconds()
				start = time.Now()
				id2 := v2Store.Commit()
				v2Time += time.Since(start).Nanoseconds()
				fmt.Printf("v1: %d, v2: %d, t1: %d, t2: %d\n", id, id2, v1Time, v2Time)
				repeat = true
			}
			return nil
		},
	}

	cmd.Flags().String(flags.FlagDst, "dst", "The migrating db dst")

	return cmd
}
