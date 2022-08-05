package tool

import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
)

func GetOffsetBySymbol(binPath, symbol string) (uint64, error) {
	f, err := os.Open(binPath)
	if err != nil {
		return 0, fmt.Errorf("open file '%s': %w", binPath, err)
	}
	defer f.Close()

	se, err := elf.NewFile(f)
	if err != nil {
		return 0, fmt.Errorf("parse ELF file: %w", err)
	}

	syms, err := se.Symbols()
	if err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		return 0, err
	}

	dynsyms, err := se.DynamicSymbols()
	if err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		return 0, err
	}
	syms = append(syms, dynsyms...)

	for _, s := range syms {
		if s.Name == symbol {
			fmt.Println("found")

			if elf.ST_TYPE(s.Info) != elf.STT_FUNC {
				// Symbol not associated with a function or other executable code.
				continue
			}

			off := s.Value

			// Loop over ELF segments.
			for _, prog := range se.Progs {
				// Skip uninteresting segments.
				if prog.Type != elf.PT_LOAD || (prog.Flags&elf.PF_X) == 0 {
					continue
				}

				if prog.Vaddr <= s.Value && s.Value < (prog.Vaddr+prog.Memsz) {
					// If the symbol value is contained in the segment, calculate
					// the symbol offset.
					//
					// fn symbol offset = fn symbol VA - .text VA + .text offset
					//
					// stackoverflow.com/a/40249502
					off = s.Value - prog.Vaddr + prog.Off
					break
				}
			}

			return off, nil
		}
	}

	return 0, fmt.Errorf("%s for %s not found", symbol, binPath)
}
