package main

import (
	"flag"
	"fmt"

	"github.com/ryngrn/droboctl/internal/device"
	"github.com/ryngrn/droboctl/internal/esa"
)

func doctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	explicit := fs.String("device", "", "SCSI generic device, e.g. /dev/sg3")
	if err := fs.Parse(args); err != nil {
		return err
	}

	d, err := device.One(*explicit)
	if err != nil {
		return err
	}

	statusRaw, err := readPage(d.SG, esa.StatusInfo, 64)
	if err != nil {
		return fmt.Errorf("status read: %w", err)
	}
	st, err := esa.ParseStatus(statusRaw)
	if err != nil {
		return err
	}

	slotRaw, err := readPage(d.SG, esa.SlotInfo2, 1308)
	if err != nil {
		return fmt.Errorf("slot read: %w", err)
	}
	slots, err := esa.ParseSlotInfo2(slotRaw)
	if err != nil {
		return err
	}

	fmt.Printf("Overall: %s\n", st.Severity())
	fmt.Printf("Status word: 0x%x\n", st.Word)
	fmt.Printf("Relayout count: %d\n", st.RelayoutCount)
	fmt.Printf("Disk pack word: 0x%x\n\n", st.DiskPackWord)

	problems := 0
	for _, s := range slots {
		if s.IsSSD() {
			if s.Health() != "Good" || s.ErrorCount > 0 {
				problems++
				fmt.Printf("Cache: %s, errors=%d, life=%d%%\n", s.Health(), s.ErrorCount, s.LifeRemaining)
			}
			continue
		}
		if s.Health() != "Good" || s.ErrorCount > 0 {
			problems++
			fmt.Printf("Bay %d: %s, errors=%d, model=%s, serial=%s\n",
				int(s.ID)+1, s.Health(), s.ErrorCount, s.Model, s.Serial)
		}
	}

	if problems == 0 {
		fmt.Println("No bay-level warnings reported by the enclosure.")
	} else {
		fmt.Printf("\n%d bay/cache warning(s) reported by the enclosure.\n", problems)
		fmt.Println("Do not remove a drive solely from this output. Confirm redundancy/rebuild state and keep an external backup before replacing hardware.")
	}
	return nil
}
