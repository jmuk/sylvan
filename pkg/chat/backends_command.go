package chat

import (
	"context"
	"fmt"

	"github.com/jmuk/sylvan/pkg/config"
	"github.com/manifoldco/promptui"
)

func (c *Chat) handleBackendsCommand(ctx context.Context, args []string) error {
	if err := c.cs.maybeInit(ctx, c.cwd); err != nil {
		return err
	}
	backendNames, err := getBackendNames(c.cs.cfg)
	if err != nil {
		return err
	}
	pos := -1
	for i, name := range backendNames {
		if name == c.cs.cfg.BackendName {
			pos = i
			break
		}
	}
	sel := promptui.Select{
		Items:     backendNames,
		CursorPos: pos,
		Size:      20,
	}
	_, selected, err := sel.Run()
	if err != nil {
		fmt.Println(err)
		return err
	}
	if selected == c.cs.cfg.BackendName {
		return nil
	}

	// Store the config file globally -- there should be a chance to customize
	// the location though.
	configFile, err := config.DefaultConfigFile()
	if err != nil {
		return err
	}
	err = config.EditConfig(configFile, func(cfg *config.Config) (*config.Config, error) {
		cfg.BackendName = selected
		return cfg, nil
	})
	if err != nil {
		return err
	}
	c.cs.ag = nil

	return c.handleModelsCommand(ctx, nil)
}
