package chat

import (
	"context"
	"fmt"

	"github.com/jmuk/sylvan/pkg/config"
	"github.com/manifoldco/promptui"
)

func (c *Chat) handleModelsCommand(ctx context.Context, args []string) error {
	if err := c.cs.maybeInit(ctx, c.cwd); err != nil {
		return err
	}
	backend, err := getBackend(c.cs.cfg)
	if err != nil {
		return err
	}
	models, err := backend.Models(ctx)
	if err != nil {
		return err
	}
	pos := 0
	for i, m := range models {
		if m == c.cs.cfg.ModelName {
			pos = i
			break
		}
	}
	sel := promptui.Select{
		Items:     models,
		CursorPos: pos,
		Size:      20,
	}
	_, selected, err := sel.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return nil
		}
		fmt.Println(err)
		return err
	}
	if selected != c.cs.cfg.ModelName {
		// Store the config file globally -- there should be a chance to customize
		// the location though.
		configFile, err := config.DefaultConfigFile()
		if err != nil {
			return err
		}
		err = config.EditConfig(configFile, func(cfg *config.Config) (*config.Config, error) {
			cfg.ModelName = selected
			return cfg, nil
		})
		if err != nil {
			return err
		}
		c.cs.ag = nil
	}
	return nil
}
