package server

// DONTCOVER

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/db/rocksdb"
	pruningtypes "github.com/cosmos/cosmos-sdk/pruning/types"
	"github.com/cosmos/cosmos-sdk/store/rootmulti"
	"github.com/cosmos/cosmos-sdk/store/v2alpha1/multi"
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

			fmt.Printf("migrated home: %s, dst: %s\n", home, dst)

			db, err := openDB(home, GetAppDBBackend(ctx.Viper))
			if err != nil {
				return err
			}

			prefix := "s/k:evm/"
			db = dbm.NewPrefixDB(db, []byte(prefix))
			cms := rootmulti.NewStore(db, ctx.Logger)

			dbSS, err := rocksdb.NewDB(dst + "_ss")
			if err != nil {
				return err
			}

			dbSC, err := rocksdb.NewDB(dst + "_sc")
			if err != nil {
				return err
			}

			storeConfig := multi.DefaultStoreConfig()
			storeConfig.Pruning = pruningtypes.NewPruningOptions(pruningtypes.PruningNothing)
			storeConfig.StateCommitmentDB = dbSC
			fmt.Printf("migrated home: %s, dst: %s\n", home, dst)
			v2Store, err := multi.MigrateFromV1(cms, dbSS, storeConfig)
			if err != nil {
				return err
			}

			err = v2Store.Close()
			if err != nil {
				return err
			}

			err = dbSS.Close()
			if err != nil {
				return err
			}

			err = dbSC.Close()
			if err != nil {
				return err
			}

			err = db.Close()
			if err != nil {
				return err
			}

			fmt.Printf("migrated from v1 to v2\n")
			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, "home", "The application home directory")
	cmd.Flags().String(flags.FlagDst, "dst", "The migrating db dst")

	return cmd
}
