package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ryngrn/droboctl/internal/device"
	"github.com/ryngrn/droboctl/internal/esa"
	"github.com/ryngrn/droboctl/internal/scsi"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "status":
		if err := status(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "drobo:", err)
			os.Exit(1)
		}
	case "doctor":
		if err := doctor(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "drobo:", err)
			os.Exit(1)
		}
	case "mount":
		if err := mountCmd(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "drobo:", err)
			os.Exit(1)
		}
	case "version":
		fmt.Println("droboctl dev")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  drobo status [--device /dev/sgX]")
	fmt.Fprintln(os.Stderr, "  drobo doctor [--device /dev/sgX] [--show-serials]")
	fmt.Fprintln(os.Stderr, "  drobo mount [--device /dev/sgX] [--mountpoint /mnt/drobo]")
}

func status(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	explicit := fs.String("device", "", "SCSI generic device, e.g. /dev/sg3")
	if err := fs.Parse(args); err != nil {
		return err
	}

	d, err := device.One(*explicit)
	if err != nil {
		return err
	}

	protoRaw, err := readPage(d.SG, esa.ProtocolVersion, 64)
	if err != nil {
		return fmt.Errorf("protocol read: %w", err)
	}
	maj, min, err := esa.ParseProtocolVersion(protoRaw)
	if err != nil {
		return err
	}

	capRaw, err := readPage(d.SG, esa.CapacityInfo, 64)
	if err != nil {
		return fmt.Errorf("capacity read: %w", err)
	}
	capacity, err := esa.ParseCapacity(capRaw)
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

	name := strings.TrimSpace(strings.TrimSpace(d.Vendor) + " " + strings.TrimSpace(d.Model))
	if name == "" {
		name = "Drobo"
	}
	fmt.Printf("%s\n", name)
	fmt.Printf("Health:   %s\n", st.Severity())
	fmt.Printf("Protocol: %d.%d\n", maj, min)
	fmt.Printf("Capacity: %s used / %s usable\n", human(capacity.UsedProtected), human(capacity.TotalProtected))
	fmt.Printf("Free:     %s\n", human(capacity.FreeProtected))
	if d.Block != "" {
		fmt.Printf("Volume:   %s\n", d.Block)
		if part, fsType, label, ferr := findDataPartition(d.Block); ferr == nil {
			if mp, ok := mountedAt(part); ok {
				fmt.Printf("Mount:    %s (%s, %s)\n", mp, fsType, label)
			} else {
				fmt.Printf("Mount:    not mounted (%s, %s on %s)\n", fsType, label, part)
			}
		}
	}
	if st.RelayoutCount > 0 {
		fmt.Printf("Relayout: %d in progress\n", st.RelayoutCount)
	}
	fmt.Println()

	for _, s := range slots {
		role := fmt.Sprintf("Bay %d", int(s.ID)+1)
		if s.IsSSD() {
			role = "Cache"
		}
		fmt.Printf("%-6s %-8s %-28s %8s", role, s.Health(), s.Model, human(s.TotalCapacity))
		if s.ErrorCount > 0 {
			fmt.Printf("  errors=%d", s.ErrorCount)
		}
		if s.IsSSD() && s.LifeRemaining > 0 {
			fmt.Printf("  life=%d%%", s.LifeRemaining)
		}
		fmt.Println()
	}
	return nil
}

func readPage(dev string, subpage byte, allocation int) ([]byte, error) {
	cdb := esa.BuildModeSense10(subpage, uint16(allocation))
	return scsi.Read(dev, cdb[:], allocation)
}

func human(n uint64) string {
	const (
		KB = 1000
		MB = KB * 1000
		GB = MB * 1000
		TB = GB * 1000
	)
	switch {
	case n >= TB:
		return fmt.Sprintf("%.2f TB", float64(n)/TB)
	case n >= GB:
		return fmt.Sprintf("%.2f GB", float64(n)/GB)
	case n >= MB:
		return fmt.Sprintf("%.2f MB", float64(n)/MB)
	default:
		return fmt.Sprintf("%d B", n)
	}
}
