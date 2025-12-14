package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
)

type Province struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type LocationConfig struct {
    Provinces []Province `json:"provinces"`
    Cities    []struct {
        ID       string `json:"id"`
        Name     string `json:"name"`
        Province string `json:"province"`
    } `json:"cities"`
}

type AssetEntry struct {
    Provinsi string   `json:"provinsi"`
    Kota     []string `json:"kota"`
}

type AssetEntryEnriched struct {
    Provinsi string            `json:"provinsi"`
    Kota     []AssetCityRecord `json:"kota"`
}

type AssetCityRecord struct {
    Name       string `json:"name"`
    ProvinceID string `json:"province_id"`
}

func main() {
    root, _ := os.Getwd()
    configPath := filepath.Join(root, "config", "location.json")
    assetPath := filepath.Join(root, "assets", "location.json")

    // Load province mapping
    cfgBytes, err := ioutil.ReadFile(configPath)
    if err != nil {
        panic(fmt.Errorf("failed to read config location.json: %w", err))
    }
    var locCfg LocationConfig
    if err := json.Unmarshal(cfgBytes, &locCfg); err != nil {
        panic(fmt.Errorf("failed to parse config location.json: %w", err))
    }
    nameToID := make(map[string]string)
    for _, p := range locCfg.Provinces {
        nameToID[strings.ToLower(strings.TrimSpace(p.Name))] = p.ID
    }

    // Load assets structure
    assetBytes, err := ioutil.ReadFile(assetPath)
    if err != nil {
        panic(fmt.Errorf("failed to read assets location.json: %w", err))
    }
    var assets []AssetEntry
    if err := json.Unmarshal(assetBytes, &assets); err != nil {
        panic(fmt.Errorf("failed to parse assets location.json: %w", err))
    }

    // Enrich with province_id
    enriched := make([]AssetEntryEnriched, 0, len(assets))
    for _, entry := range assets {
        provLower := strings.ToLower(strings.TrimSpace(entry.Provinsi))
        provID := nameToID[provLower]
        cities := make([]AssetCityRecord, 0, len(entry.Kota))
        for _, c := range entry.Kota {
            cities = append(cities, AssetCityRecord{
                Name:       strings.TrimSpace(c),
                ProvinceID: provID,
            })
        }
        enriched = append(enriched, AssetEntryEnriched{
            Provinsi: entry.Provinsi,
            Kota:     cities,
        })
    }

    // Write back enriched file
    outBytes, err := json.MarshalIndent(enriched, "", "  ")
    if err != nil {
        panic(fmt.Errorf("failed to marshal enriched assets: %w", err))
    }
    if err := ioutil.WriteFile(assetPath, outBytes, 0644); err != nil {
        panic(fmt.Errorf("failed to write enriched assets: %w", err))
    }
}

