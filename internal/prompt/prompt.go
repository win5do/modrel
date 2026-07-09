package prompt

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/win5do/modrel/internal/discovery"
	"github.com/win5do/modrel/internal/version"
)

func SelectModule(modules []discovery.Module) (discovery.Module, error) {
	if len(modules) == 0 {
		return discovery.Module{}, fmt.Errorf("no Go modules discovered")
	}

	options := make([]huh.Option[string], 0, len(modules))
	for _, module := range modules {
		label := fmt.Sprintf("%s  %s", module.Name, module.ModulePath)
		options = append(options, huh.NewOption(label, module.Name))
	}

	selected := modules[0].Name
	err := huh.NewSelect[string]().
		Title("Select module").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return discovery.Module{}, err
	}

	for _, module := range modules {
		if module.Name == selected {
			return module, nil
		}
	}
	return discovery.Module{}, fmt.Errorf("selected module %q was not discovered", selected)
}

func SelectVersionMode() (releaseType string, manualVersion string, err error) {
	const manual = "manual"

	selected := "stable"
	err = huh.NewSelect[string]().
		Title("Select release type").
		Options(
			huh.NewOption("Stable", "stable"),
			huh.NewOption("RC", "rc"),
			huh.NewOption("Manual", manual),
		).
		Value(&selected).
		Run()
	if err != nil {
		return "", "", err
	}

	if selected != manual {
		return selected, "", nil
	}

	err = huh.NewInput().
		Title("Enter release version").
		Placeholder("v1.2.3 or v1.2.3-rc.1").
		Value(&manualVersion).
		Validate(version.Validate).
		Run()
	if err != nil {
		return "", "", err
	}
	return manual, manualVersion, nil
}
