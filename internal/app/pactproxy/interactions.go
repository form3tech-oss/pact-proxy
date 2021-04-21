package pactproxy

import (
	"sync"
)

type Interactions struct {
	interactions sync.Map
}

func (i *Interactions) Store(interaction *interaction) {
	i.interactions.Store(interaction.Description, interaction)
	if interaction.Alias != "" {
		i.interactions.Store(interaction.Alias, interaction)
	}
}

func (i *Interactions) Clear() {
	i.interactions.Range(func(k, _ interface{}) bool {
		i.interactions.Delete(k)
		return true
	})
}

func (i *Interactions) Load(key string) (*interaction, bool) {
	result, ok := i.interactions.Load(key)
	if !ok {
		return nil, false
	}
	return result.(*interaction), true
}

func (i *Interactions) FindAll(path, method string) ([]*interaction, bool) {
	interactions := make(map[string]*interaction)
	var result []*interaction
	i.interactions.Range(func(_, v interface{}) bool {
		if v.(*interaction).Match(path, method) {
			i := v.(*interaction)
			interactions[i.Description] = i
		}
		return true
	})

	for _, i := range interactions {
		result = append(result, i)
	}

	return result, len(result) > 0
}

func (i *Interactions) All() []*interaction {
	var interactions []*interaction
	i.interactions.Range(func(_, v interface{}) bool {
		interactions = append(interactions, v.(*interaction))
		return true
	})
	return interactions
}

func (i *Interactions) AllHaveRequests() bool {
	result := true
	i.interactions.Range(func(_, v interface{}) bool {
		if !v.(*interaction).HasRequests(1) {
			result = false
			return false
		}
		return true
	})
	return result
}
