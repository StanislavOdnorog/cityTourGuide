package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type seedPOI struct {
	Name          string
	NameRu        string
	Lat           float64
	Lng           float64
	Type          string
	Address       string
	InterestScore int16
	StoriesEN     []seedStory
	StoriesRU     []seedStory
}

type seedStory struct {
	Text      string
	LayerType string
	Duration  int16
}

func tbilisiPOIs() []seedPOI {
	return []seedPOI{
		{
			Name: "Narikala Fortress", NameRu: "Крепость Нарикала",
			Lat: 41.6875, Lng: 44.8089, Type: "monument",
			Address: "Narikala, Old Tbilisi", InterestScore: 80,
			StoriesEN: []seedStory{{
				Text:      "Narikala Fortress has watched over Tbilisi since the 4th century. Originally built by the Persians, it was later expanded by the Arabs, Mongols, and Georgians. The fortress suffered its greatest damage not from any army, but from a Russian ammunition depot explosion in 1827. Today its weathered walls frame the best panoramic views of the city, and the restored St. Nicholas Church within its walls still holds services.",
				LayerType: "atmosphere", Duration: 45,
			}},
			StoriesRU: []seedStory{{
				Text:      "Крепость Нарикала наблюдает за Тбилиси с IV века. Изначально построенная персами, она была расширена арабами, монголами и грузинами. Наибольший ущерб крепости нанесла не армия, а взрыв русского порохового склада в 1827 году. Сегодня её выветренные стены обрамляют лучшие панорамные виды города, а восстановленная церковь Святого Николая в её стенах до сих пор проводит службы.",
				LayerType: "atmosphere", Duration: 48,
			}},
		},
		{
			Name: "Abanotubani (Sulfur Baths)", NameRu: "Абанотубани (Серные бани)",
			Lat: 41.6879, Lng: 44.8103, Type: "district",
			Address: "Abanotubani, Old Tbilisi", InterestScore: 75,
			StoriesEN: []seedStory{{
				Text:      "The legend says King Vakhtang Gorgasali's falcon chased a pheasant into a hot spring here, and the king was so impressed he moved his capital to this spot. Whether or not a bird founded the city, these sulfur baths have been the social heart of Tbilisi for centuries. The distinctive brick domes you see cover pools where poets, kings, and ordinary people have soaked side by side. Pushkin himself declared he had never experienced anything more luxurious.",
				LayerType: "human_story", Duration: 52,
			}},
			StoriesRU: []seedStory{{
				Text:      "Легенда гласит, что сокол царя Вахтанга Горгасали погнался за фазаном прямо в горячий источник, и царь был так впечатлён, что перенёс столицу на это место. Независимо от того, основала ли птица город, эти серные бани были социальным сердцем Тбилиси на протяжении веков. Характерные кирпичные купола, которые вы видите, укрывают бассейны, где поэты, цари и обычные люди парились бок о бок. Сам Пушкин заявил, что никогда не испытывал ничего более роскошного.",
				LayerType: "human_story", Duration: 55,
			}},
		},
		{
			Name: "Bridge of Peace", NameRu: "Мост Мира",
			Lat: 41.6934, Lng: 44.8095, Type: "bridge",
			Address: "Bridge of Peace, Tbilisi", InterestScore: 70,
			StoriesEN: []seedStory{{
				Text:      "This glass-and-steel pedestrian bridge, designed by Italian architect Michele De Lucchi, opened in 2010 and quickly became one of Tbilisi's most recognizable landmarks. Its undulating canopy of glass is fitted with thousands of LED lights that display an interactive light show each evening. The bridge connects the Old Town with Rike Park and symbolizes Georgia's connection between its ancient past and modern future.",
				LayerType: "hidden_detail", Duration: 40,
			}},
			StoriesRU: []seedStory{{
				Text:      "Этот стеклянно-стальной пешеходный мост, спроектированный итальянским архитектором Микеле Де Лукки, был открыт в 2010 году и быстро стал одной из самых узнаваемых достопримечательностей Тбилиси. Его волнообразный стеклянный навес оснащён тысячами светодиодов, которые каждый вечер устраивают интерактивное световое шоу. Мост соединяет Старый город с парком Рике и символизирует связь Грузии между её древним прошлым и современным будущим.",
				LayerType: "hidden_detail", Duration: 43,
			}},
		},
		{
			Name: "Rustaveli Avenue", NameRu: "Проспект Руставели",
			Lat: 41.7017, Lng: 44.7934, Type: "street",
			Address: "Rustaveli Ave, Tbilisi", InterestScore: 65,
			StoriesEN: []seedStory{{
				Text:      "Named after the medieval Georgian poet Shota Rustaveli, this grand boulevard is the political and cultural spine of Tbilisi. Walking along it, you pass the Parliament building, the National Museum, the Opera House, and the Academy of Sciences. The avenue has witnessed some of Georgia's most dramatic moments — from the 1989 Soviet crackdown to the 2003 Rose Revolution. The wide plane-tree-lined sidewalks buzz with cafes and street performers.",
				LayerType: "time_shift", Duration: 50,
			}},
			StoriesRU: []seedStory{{
				Text:      "Названный в честь средневекового грузинского поэта Шота Руставели, этот величественный бульвар является политическим и культурным хребтом Тбилиси. Прогуливаясь по нему, вы проходите мимо здания Парламента, Национального музея, Оперного театра и Академии наук. Проспект был свидетелем самых драматичных моментов Грузии — от советской расправы 1989 года до Революции роз 2003 года. Широкие тротуары, обсаженные платанами, кипят кафе и уличными артистами.",
				LayerType: "time_shift", Duration: 53,
			}},
		},
		{
			Name: "Holy Trinity Cathedral (Sameba)", NameRu: "Собор Святой Троицы (Цминда Самеба)",
			Lat: 41.6976, Lng: 44.8166, Type: "church",
			Address: "Sameba Cathedral, Tbilisi", InterestScore: 75,
			StoriesEN: []seedStory{{
				Text:      "The Holy Trinity Cathedral, locally known as Sameba, is the largest religious building in the South Caucasus. Completed in 2004 after nearly a decade of construction, it stands 84 meters tall on St. Ilia Hill. The cathedral was a project of national significance — funded by private donations and the Georgian Orthodox Church. Its golden dome is visible from almost every point in the city, serving as a constant landmark for orientation.",
				LayerType: "general", Duration: 45,
			}},
			StoriesRU: []seedStory{{
				Text:      "Собор Святой Троицы, известный местным как Цминда Самеба, является крупнейшим религиозным сооружением на Южном Кавказе. Завершённый в 2004 году после почти десятилетия строительства, он возвышается на 84 метра на холме Святого Ильи. Собор был проектом национального значения — финансировался частными пожертвованиями и Грузинской Православной Церковью. Его золотой купол виден практически из любой точки города, служа постоянным ориентиром.",
				LayerType: "general", Duration: 48,
			}},
		},
		{
			Name: "Tbilisi Botanical Garden", NameRu: "Тбилисский ботанический сад",
			Lat: 41.6857, Lng: 44.8131, Type: "park",
			Address: "Botanical Garden, Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "Nestled behind Narikala Fortress in a deep gorge carved by the Tsavkisis-Tskali river, the Tbilisi Botanical Garden has been a green refuge since the early 1800s. The garden features a stunning waterfall that drops into a natural pool, surrounded by centuries-old trees from across the Caucasus. In Soviet times, scientists here preserved rare plant species from throughout the empire. Today it remains one of the quietest spots in the city center.",
				LayerType: "atmosphere", Duration: 42,
			}},
			StoriesRU: []seedStory{{
				Text:      "Расположенный за крепостью Нарикала в глубоком ущелье, вырезанном рекой Цавкисис-Цкали, Тбилисский ботанический сад является зелёным убежищем с начала 1800-х годов. Сад украшает потрясающий водопад, падающий в природный бассейн, окружённый вековыми деревьями со всего Кавказа. В советское время учёные здесь сохраняли редкие виды растений со всей империи. Сегодня это одно из самых тихих мест в центре города.",
				LayerType: "atmosphere", Duration: 45,
			}},
		},
		{
			Name: "Freedom Square", NameRu: "Площадь Свободы",
			Lat: 41.6941, Lng: 44.8015, Type: "square",
			Address: "Freedom Square, Tbilisi", InterestScore: 65,
			StoriesEN: []seedStory{{
				Text:      "Freedom Square has been the beating heart of Georgian political life under many names. Under the Russian Empire it was Erivan Square, the Soviets renamed it Lenin Square, and after independence in 1991 it became Freedom Square. The tall column in the center is topped by a golden statue of St. George slaying the dragon, created by the celebrated Georgian sculptor Zurab Tsereteli. Underground, one of Tbilisi's busiest metro stations connects the old and new parts of the city.",
				LayerType: "time_shift", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Площадь Свободы была бьющимся сердцем грузинской политической жизни под многими именами. При Российской империи она называлась Эриванской площадью, Советы переименовали её в площадь Ленина, а после обретения независимости в 1991 году она стала площадью Свободы. Высокую колонну в центре венчает золотая статуя Святого Георгия, поражающего дракона, созданная знаменитым грузинским скульптором Зурабом Церетели. Под землёй одна из самых загруженных станций метро Тбилиси соединяет старую и новую части города.",
				LayerType: "time_shift", Duration: 52,
			}},
		},
		{
			Name: "Georgian National Museum", NameRu: "Грузинский национальный музей",
			Lat: 41.6989, Lng: 44.7989, Type: "museum",
			Address: "3 Rustaveli Ave, Tbilisi", InterestScore: 70,
			StoriesEN: []seedStory{{
				Text:      "The Georgian National Museum on Rustaveli Avenue houses treasures spanning millions of years. Its most famous collection is the Trialeti Gold — a set of exquisite gold goblets and jewelry from the 2nd millennium BC that rival anything found in ancient Egypt or Mesopotamia. The museum also holds Dmanisi skulls, the oldest human remains found outside Africa, dating back 1.8 million years. These discoveries forced scientists to rewrite the story of human migration.",
				LayerType: "human_story", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Грузинский национальный музей на проспекте Руставели хранит сокровища, охватывающие миллионы лет. Его самая знаменитая коллекция — Триалетское золото, набор изысканных золотых кубков и украшений II тысячелетия до нашей эры, не уступающих находкам Древнего Египта или Месопотамии. Музей также хранит Дманисские черепа — древнейшие человеческие останки, найденные за пределами Африки, возрастом 1,8 миллиона лет. Эти открытия заставили учёных переписать историю миграции человека.",
				LayerType: "human_story", Duration: 50,
			}},
		},
		{
			Name: "Sioni Cathedral", NameRu: "Сионский кафедральный собор",
			Lat: 41.6917, Lng: 44.8079, Type: "church",
			Address: "Sioni St, Tbilisi", InterestScore: 65,
			StoriesEN: []seedStory{{
				Text:      "Sioni Cathedral, named after Mount Zion in Jerusalem, has been the spiritual center of Georgia since the 6th century. The current building dates primarily from the 13th century, though it has been destroyed and rebuilt multiple times by invading armies. Inside, it houses the Cross of St. Nino — the most sacred relic of the Georgian Orthodox Church. According to tradition, St. Nino made the cross from grapevines bound with her own hair when she brought Christianity to Georgia in the 4th century.",
				LayerType: "general", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Сионский собор, названный в честь горы Сион в Иерусалиме, является духовным центром Грузии с VI века. Нынешнее здание относится преимущественно к XIII веку, хотя оно неоднократно разрушалось и перестраивалось захватчиками. Внутри хранится Крест Святой Нино — самая священная реликвия Грузинской Православной Церкви. Согласно преданию, Святая Нино сделала крест из виноградных лоз, связанных собственными волосами, когда принесла христианство в Грузию в IV веке.",
				LayerType: "general", Duration: 52,
			}},
		},
		{
			Name: "Mtatsminda Park", NameRu: "Парк Мтацминда",
			Lat: 41.6934, Lng: 44.7867, Type: "park",
			Address: "Mtatsminda, Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "Perched atop Mtatsminda Mountain at 770 meters above sea level, this amusement park offers the highest viewpoint in Tbilisi. The funicular railway that climbs the mountain has been running since 1905, making it one of the oldest urban cable railways in the former Soviet Union. On clear days, you can see the snowy peaks of the Greater Caucasus from the observation deck. Below the park lies the Mtatsminda Pantheon, where Georgia's greatest writers, artists, and national heroes rest.",
				LayerType: "hidden_detail", Duration: 44,
			}},
			StoriesRU: []seedStory{{
				Text:      "Расположенный на вершине горы Мтацминда на высоте 770 метров над уровнем моря, этот парк развлечений предлагает самую высокую смотровую площадку в Тбилиси. Фуникулёр, поднимающийся на гору, работает с 1905 года, что делает его одной из старейших городских канатных дорог на территории бывшего СССР. В ясные дни с обзорной площадки видны снежные вершины Большого Кавказа. Под парком находится Пантеон Мтацминда, где покоятся величайшие писатели, художники и национальные герои Грузии.",
				LayerType: "hidden_detail", Duration: 48,
			}},
		},
	}
}

func ensureTbilisiCity(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var cityID int
	err := pool.QueryRow(ctx, `SELECT id FROM cities WHERE name = $1`, "Tbilisi").Scan(&cityID)
	if err == nil {
		log.Printf("Tbilisi already exists (id=%d), skipping city insert", cityID)
		return cityID, nil
	}

	nameRu := "Тбилиси"
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (name, name_ru, country, center_lat, center_lng, radius_km, is_active, download_size_mb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		"Tbilisi", &nameRu, "Georgia", 41.7151, 44.8271, 10.0, true, 0.0,
	).Scan(&cityID)
	if err != nil {
		return 0, fmt.Errorf("insert Tbilisi: %w", err)
	}

	log.Printf("Created Tbilisi (id=%d)", cityID)
	return cityID, nil
}

func poiExistsByName(ctx context.Context, pool *pgxpool.Pool, cityID int, name string) (int, bool) {
	var poiID int
	err := pool.QueryRow(ctx, `SELECT id FROM poi WHERE city_id = $1 AND name = $2`, cityID, name).Scan(&poiID)
	if err != nil {
		return 0, false
	}
	return poiID, true
}

func insertPOI(ctx context.Context, pool *pgxpool.Pool, cityID int, p *seedPOI) (int, error) {
	tags, _ := json.Marshal(map[string]string{"source": "seed"})

	var poiID int
	err := pool.QueryRow(ctx, `
		INSERT INTO poi (city_id, name, name_ru, location, type, tags, address, interest_score, status)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography, $6, $7, $8, $9, 'active')
		RETURNING id`,
		cityID, p.Name, &p.NameRu, p.Lng, p.Lat, p.Type, tags, &p.Address, p.InterestScore,
	).Scan(&poiID)
	if err != nil {
		return 0, fmt.Errorf("insert poi %q: %w", p.Name, err)
	}
	return poiID, nil
}

func storyExists(ctx context.Context, pool *pgxpool.Pool, poiID int, lang string) bool {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM story WHERE poi_id = $1 AND language = $2)`, poiID, lang).Scan(&exists)
	return err == nil && exists
}

func insertStory(ctx context.Context, pool *pgxpool.Pool, poiID int, lang string, s seedStory) error {
	fakeAudioURL := fmt.Sprintf("https://example.com/audio/seed/%d_%s.mp3", poiID, lang)
	sources, _ := json.Marshal([]string{"seed_data"})

	_, err := pool.Exec(ctx, `
		INSERT INTO story (poi_id, language, text, audio_url, duration_sec, layer_type, order_index, is_inflation, confidence, sources, status)
		VALUES ($1, $2, $3, $4, $5, $6, 0, false, 90, $7, 'active')`,
		poiID, lang, s.Text, fakeAudioURL, s.Duration, s.LayerType, sources,
	)
	if err != nil {
		return fmt.Errorf("insert story for poi %d (%s): %w", poiID, lang, err)
	}
	return nil
}

func run() error {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	if pingErr := pool.Ping(ctx); pingErr != nil {
		return fmt.Errorf("ping database: %w", pingErr)
	}
	log.Println("Database connected")

	cityID, err := ensureTbilisiCity(ctx, pool)
	if err != nil {
		return err
	}

	pois := tbilisiPOIs()
	var poisCreated, poisSkipped, storiesCreated, storiesSkipped int

	for i := range pois {
		p := &pois[i]
		poiID, exists := poiExistsByName(ctx, pool, cityID, p.Name)
		if exists {
			poisSkipped++
			log.Printf("  POI %q already exists (id=%d), skipping", p.Name, poiID)
		} else {
			poiID, err = insertPOI(ctx, pool, cityID, p)
			if err != nil {
				log.Printf("  ERROR: %v", err)
				continue
			}
			poisCreated++
			log.Printf("  Created POI %q (id=%d)", p.Name, poiID)
		}

		for _, s := range p.StoriesEN {
			if storyExists(ctx, pool, poiID, "en") {
				storiesSkipped++
				continue
			}
			if sErr := insertStory(ctx, pool, poiID, "en", s); sErr != nil {
				log.Printf("  ERROR: %v", sErr)
				continue
			}
			storiesCreated++
		}

		for _, s := range p.StoriesRU {
			if storyExists(ctx, pool, poiID, "ru") {
				storiesSkipped++
				continue
			}
			if sErr := insertStory(ctx, pool, poiID, "ru", s); sErr != nil {
				log.Printf("  ERROR: %v", sErr)
				continue
			}
			storiesCreated++
		}
	}

	fmt.Println()
	fmt.Println("=== Seed Summary ===")
	fmt.Printf("  POIs created:    %d\n", poisCreated)
	fmt.Printf("  POIs skipped:    %d\n", poisSkipped)
	fmt.Printf("  Stories created: %d\n", storiesCreated)
	fmt.Printf("  Stories skipped: %d\n", storiesSkipped)
	fmt.Printf("  Total POIs:      %d\n", len(pois))
	fmt.Printf("  Total stories:   %d (10 EN + 10 RU)\n", len(pois)*2)

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
