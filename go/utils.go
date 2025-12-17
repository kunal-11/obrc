package main

func mergeMaps(maps []map[string]*CityTotal) map[string]*CityTotal {
	result := make(map[string]*CityTotal)
	for _, m := range maps {
		for k, v := range m {
			if cur, ok := result[k]; ok {
				cur.Count += v.Count
				cur.Sum += v.Sum
				cur.Max = max(cur.Max, v.Max)
				cur.Min = min(cur.Min, v.Min)
			} else {
				result[k] = v
			}
		}
	}
	return result
}
