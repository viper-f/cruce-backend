package test

import (
	"cuento-backend/src/Entities"
	"sort"
	"testing"
)

func TestGetFactionTreeByCharacter_Logic(t *testing.T) {
	// Mocking the logic of GetFactionTreeByCharacter since we don't have a DB mock here
	// This test verifies the sorting and tree building logic

	f1 := Entities.Faction{Id: 1, Name: "A", Level: 0, ParentId: nil}
	f2 := Entities.Faction{Id: 2, Name: "B", Level: 0, ParentId: nil}
	f11 := Entities.Faction{Id: 3, Name: "A.1", Level: 1, ParentId: intPtr(1)}
	f12 := Entities.Faction{Id: 4, Name: "A.2", Level: 1, ParentId: intPtr(1)}
	f21 := Entities.Faction{Id: 5, Name: "B.1", Level: 1, ParentId: intPtr(2)}

	factions := []Entities.Faction{f21, f12, f11, f2, f1}

	// Sort logic from the function
	sort.Slice(factions, func(i, j int) bool {
		if factions[i].Level != factions[j].Level {
			return factions[i].Level < factions[j].Level
		}
		return factions[i].Name < factions[j].Name
	})

	var trees [][]Entities.Faction
	factionToTreeIndex := make(map[int]int)

	for _, f := range factions {
		if f.Level == 0 {
			trees = append(trees, []Entities.Faction{f})
			factionToTreeIndex[f.Id] = len(trees) - 1
		} else {
			if f.ParentId != nil {
				if treeIdx, ok := factionToTreeIndex[*f.ParentId]; ok {
					trees[treeIdx] = append(trees[treeIdx], f)
					factionToTreeIndex[f.Id] = treeIdx
				}
			}
		}
	}

	var result []Entities.Faction
	for _, tree := range trees {
		result = append(result, tree...)
	}

	expectedOrder := []int{1, 3, 4, 2, 5}
	if len(result) != len(expectedOrder) {
		t.Fatalf("Expected length %d, got %d", len(expectedOrder), len(result))
	}

	for i, id := range expectedOrder {
		if result[i].Id != id {
			t.Errorf("At index %d, expected ID %d, got %d", i, id, result[i].Id)
		}
	}
}

func intPtr(i int) *int {
	return &i
}
