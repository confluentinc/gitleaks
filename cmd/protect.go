package cmd

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/zricethezav/gitleaks/v8/config"
	"github.com/zricethezav/gitleaks/v8/detect"
	"github.com/zricethezav/gitleaks/v8/report"
)

func init() {
	protectCmd.Flags().Bool("staged", false, "detect secrets in a --staged state")
	rootCmd.AddCommand(protectCmd)
}

var protectCmd = &cobra.Command{
	Use:   "protect",
	Short: "protect secrets in code",
	Run:   runProtect,
}

func runProtect(cmd *cobra.Command, args []string) {
	initConfig()
	var vc config.ViperConfig

	if err := viper.Unmarshal(&vc); err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}
	cfg, err := vc.Translate()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	cfg.Path, _ = cmd.Flags().GetString("config")
	exitCode, _ := cmd.Flags().GetInt("exit-code")
	staged, _ := cmd.Flags().GetBool("staged")
	start := time.Now()

	// Setup detector
	detector := detect.NewDetector(cfg)
	detector.Config.Path, err = cmd.Flags().GetString("config")
	if err != nil {
		log.Fatal().Err(err)
	}
	source, err := cmd.Flags().GetString("source")
	if err != nil {
		log.Fatal().Err(err)
	}
	// if config path is not set, then use the {source}/.gitleaks.toml path.
	// note that there may not be a `{source}/.gitleaks.toml` file, this is ok.
	if detector.Config.Path == "" {
		detector.Config.Path = filepath.Join(source, ".gitleaks.toml")
	}
	// set verbose flag
	if detector.Verbose, err = cmd.Flags().GetBool("verbose"); err != nil {
		log.Fatal().Err(err)
	}
	// set redact flag
	if detector.Redact, err = cmd.Flags().GetBool("redact"); err != nil {
		log.Fatal().Err(err)
	}

	// get log options for git scan
	logOpts, err := cmd.Flags().GetString("log-opts")
	if err != nil {
		log.Fatal().Err(err)
	}

	// start git scan
	var findings []report.Finding
	if staged {
		findings, err = detector.DetectGit(source, logOpts, detect.ProtectStagedType)
	} else {
		findings, err = detector.DetectGit(source, logOpts, detect.ProtectType)
	}
	if err != nil {
		// don't exit on error, just log it
		log.Error().Err(err)
	}

	// log info about the scan
	log.Info().Msgf("scan completed in %s", time.Since(start))
	if len(findings) != 0 {
		log.Warn().Msgf("leaks found: %d", len(findings))
	} else {
		log.Info().Msg("no leaks found")
	}

	reportPath, _ := cmd.Flags().GetString("report-path")
	ext, _ := cmd.Flags().GetString("report-format")
	if reportPath != "" {
		if err = report.Write(findings, cfg, ext, reportPath); err != nil {
			log.Fatal().Err(err)
		}
	}
	if len(findings) != 0 {
		os.Exit(exitCode)
	}
}
