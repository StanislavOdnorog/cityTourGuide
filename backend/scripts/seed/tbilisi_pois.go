package main

// tbilisiPOIs returns 55 real Tbilisi landmarks with bilingual stories (EN + RU).
// Each POI has at least one English and one Russian story, totalling 110+ stories.
// Stories follow the narrative guidelines: anchor, hook, facts, meaning.
// All coordinates are accurate for the actual locations.
func tbilisiPOIs() []seedPOI {
	return []seedPOI{
		// --- 1. Narikala Fortress ---
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
		// --- 2. Abanotubani (Sulfur Baths) ---
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
		// --- 3. Bridge of Peace ---
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
		// --- 4. Rustaveli Avenue ---
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
		// --- 5. Holy Trinity Cathedral (Sameba) ---
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
		// --- 6. Tbilisi Botanical Garden ---
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
		// --- 7. Freedom Square ---
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
		// --- 8. Georgian National Museum ---
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
		// --- 9. Sioni Cathedral ---
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
		// --- 10. Mtatsminda Park ---
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

		// =============================================
		// NEW POIs (11-55) — expanding to 55 total
		// =============================================

		// --- 11. Metekhi Church ---
		{
			Name: "Metekhi Church", NameRu: "Церковь Метехи",
			Lat: 41.6908, Lng: 44.8112, Type: "church",
			Address: "Metekhi St, Old Tbilisi", InterestScore: 70,
			StoriesEN: []seedStory{{
				Text:      "Metekhi Church stands on a cliff above the Mtkvari River, one of the most photographed spots in Tbilisi. Built in the 13th century by King Demetre II, it marks the site where Georgia's patron saint, Queen Shushanik, was martyred in the 5th century. The equestrian statue of King Vakhtang Gorgasali in front gazes across the river toward the city he founded. During Soviet times, the church served as a theater, then a prison — only returning to religious use in 1988.",
				LayerType: "time_shift", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Церковь Метехи стоит на скале над рекой Мтквари — одно из самых фотографируемых мест Тбилиси. Построенная в XIII веке царём Деметре II, она расположена на месте, где в V веке приняла мученическую смерть покровительница Грузии царица Шушаник. Конная статуя царя Вахтанга Горгасали перед церковью смотрит через реку на город, который он основал. В советское время церковь служила театром, затем тюрьмой — богослужения возобновились лишь в 1988 году.",
				LayerType: "time_shift", Duration: 50,
			}},
		},
		// --- 12. Clock Tower (Rezo Gabriadze) ---
		{
			Name: "Clock Tower of Rezo Gabriadze", NameRu: "Часовая башня Резо Габриадзе",
			Lat: 41.6921, Lng: 44.8068, Type: "monument",
			Address: "Shavteli St 13, Old Tbilisi", InterestScore: 72,
			StoriesEN: []seedStory{{
				Text:      "This crooked, fairy-tale clock tower leans over Shavteli Street as if it might topple at any moment. Built in 2010 by the legendary Georgian puppeteer and filmmaker Rezo Gabriadze, the tower is covered in tiles, each one unique. Every hour, a small angel emerges from the top door, rings a bell, and retreats. At noon and 7 PM, a miniature puppet theater plays out a love story. It is said Gabriadze placed his own philosophy into the tower — that beauty does not need to be perfect.",
				LayerType: "hidden_detail", Duration: 50,
			}},
			StoriesRU: []seedStory{{
				Text:      "Эта кривая сказочная часовая башня нависает над улицей Шавтели, словно вот-вот упадёт. Построенная в 2010 году легендарным грузинским кукольником и кинорежиссёром Резо Габриадзе, башня покрыта плитками, каждая из которых уникальна. Каждый час маленький ангел выходит из верхней двери, звонит в колокол и скрывается. В полдень и в 7 вечера миниатюрный кукольный театр разыгрывает историю любви. Говорят, Габриадзе вложил в башню свою философию — красота не обязана быть совершенной.",
				LayerType: "hidden_detail", Duration: 53,
			}},
		},
		// --- 13. Anchiskhati Basilica ---
		{
			Name: "Anchiskhati Basilica", NameRu: "Базилика Анчисхати",
			Lat: 41.6930, Lng: 44.8062, Type: "church",
			Address: "Shavteli St 19, Old Tbilisi", InterestScore: 68,
			StoriesEN: []seedStory{{
				Text:      "Anchiskhati Basilica is the oldest surviving church in Tbilisi, dating back to the 6th century. Its name comes from a precious icon — the Anchiskhati icon of Christ — that was brought here from the Anchi Fortress in southern Georgia. The stone walls you see have survived Arab raids, Mongol invasions, and Persian sieges. Inside, the church is intimate and dim, with frescoes that have faded over fifteen centuries into ghostly outlines that somehow feel more sacred for their age.",
				LayerType: "atmosphere", Duration: 44,
			}},
			StoriesRU: []seedStory{{
				Text:      "Базилика Анчисхати — старейшая сохранившаяся церковь в Тбилиси, датируемая VI веком. Своё название она получила от драгоценной иконы — Анчисхатского образа Спасителя, привезённого сюда из крепости Анчи в южной Грузии. Каменные стены, которые вы видите, пережили арабские набеги, монгольские вторжения и персидские осады. Внутри церковь камерная и тёмная, с фресками, поблёкшими за пятнадцать веков до призрачных контуров, которые каким-то образом ощущаются ещё более священными из-за своего возраста.",
				LayerType: "atmosphere", Duration: 47,
			}},
		},
		// --- 14. Leghvtakhevi Waterfall ---
		{
			Name: "Leghvtakhevi Waterfall", NameRu: "Водопад Легвтахеви",
			Lat: 41.6873, Lng: 44.8098, Type: "park",
			Address: "Leghvtakhevi, Old Tbilisi", InterestScore: 65,
			StoriesEN: []seedStory{{
				Text:      "Hidden in a narrow gorge just steps from the sulfur baths, the Leghvtakhevi waterfall is one of Tbilisi's best-kept secrets. The name means 'fig gorge' in Georgian, after the wild fig trees that grow along its walls. A short walking path takes you through a canyon barely ten meters wide, where the sound of rushing water drowns out the city above. The waterfall drops about twenty meters into a pool where children swim in summer. It is hard to believe this wild ravine exists in the middle of a capital city.",
				LayerType: "atmosphere", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Спрятанный в узком ущелье в нескольких шагах от серных бань, водопад Легвтахеви — один из самых сокровенных секретов Тбилиси. Название переводится как «инжирное ущелье» — по диким инжирным деревьям, растущим вдоль его стен. Короткая пешеходная тропа ведёт через каньон шириной едва десять метров, где шум воды заглушает город наверху. Водопад падает примерно с двадцати метров в бассейн, где летом купаются дети. Трудно поверить, что это дикое ущелье существует в центре столицы.",
				LayerType: "atmosphere", Duration: 49,
			}},
		},
		// --- 15. King Erekle II Square (Meidan) ---
		{
			Name: "Meidan Square", NameRu: "Площадь Мейдан",
			Lat: 41.6899, Lng: 44.8087, Type: "square",
			Address: "Meidan Square, Old Tbilisi", InterestScore: 62,
			StoriesEN: []seedStory{{
				Text:      "Meidan Square, officially named after King Erekle II, is the crossroads of Old Tbilisi. For centuries, this was the city's main market — 'meidan' comes from the Persian word for square or gathering place. Caravans along the Silk Road stopped here, and merchants from Persia, Turkey, Armenia, and India traded goods under its awnings. Today the square is smaller than it once was, but it still connects the sulfur baths, the mosque, the synagogue, and Armenian churches — a map of Tbilisi's multicultural identity in a single block.",
				LayerType: "human_story", Duration: 50,
			}},
			StoriesRU: []seedStory{{
				Text:      "Площадь Мейдан, официально названная в честь царя Ираклия II, является перекрёстком Старого Тбилиси. На протяжении веков здесь был главный рынок города — слово «мейдан» происходит от персидского «площадь» или «место собрания». Караваны Шёлкового пути останавливались здесь, и купцы из Персии, Турции, Армении и Индии торговали под навесами. Сегодня площадь меньше, чем была когда-то, но она по-прежнему соединяет серные бани, мечеть, синагогу и армянские церкви — карта мультикультурной идентичности Тбилиси в одном квартале.",
				LayerType: "human_story", Duration: 53,
			}},
		},
		// --- 16. Tbilisi Mosque (Juma Mosque) ---
		{
			Name: "Juma Mosque", NameRu: "Мечеть Джума",
			Lat: 41.6893, Lng: 44.8085, Type: "church",
			Address: "Botanikuri St 32, Old Tbilisi", InterestScore: 63,
			StoriesEN: []seedStory{{
				Text:      "The Juma Mosque is the only functioning mosque in Tbilisi and a remarkable symbol of the city's tolerance. Built in the 18th century, it serves both Sunni and Shia Muslims — a rarity in the Islamic world. The two communities pray in the same hall, simply facing different walls. Standing between the Armenian church and the synagogue, the mosque completes an interfaith triangle that locals proudly point to as proof that Tbilisi has always been a city where different faiths coexist.",
				LayerType: "human_story", Duration: 44,
			}},
			StoriesRU: []seedStory{{
				Text:      "Мечеть Джума — единственная действующая мечеть в Тбилиси и замечательный символ толерантности города. Построенная в XVIII веке, она служит и суннитам, и шиитам — редкость в исламском мире. Две общины молятся в одном зале, просто обращаясь к разным стенам. Расположенная между армянской церковью и синагогой, мечеть завершает межконфессиональный треугольник, на который местные жители с гордостью указывают как на доказательство того, что Тбилиси всегда был городом, где разные религии сосуществуют.",
				LayerType: "human_story", Duration: 47,
			}},
		},
		// --- 17. Tbilisi Great Synagogue ---
		{
			Name: "Tbilisi Great Synagogue", NameRu: "Большая синагога Тбилиси",
			Lat: 41.6901, Lng: 44.8071, Type: "church",
			Address: "Leselidze St 45, Old Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "The Great Synagogue of Tbilisi was built in 1903 by Georgian Jews who had lived in the region for over 2,600 years — one of the oldest Jewish communities in the world. They trace their arrival to the Babylonian captivity. The synagogue's warm brick facade and modest size belie its importance. Georgia is one of the rare countries where antisemitism never took root. It is said that when Jews arrived in ancient Colchis, they were simply told to find a spot and settle in. They did, and they stayed.",
				LayerType: "human_story", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Большая синагога Тбилиси была построена в 1903 году грузинскими евреями, жившими в регионе более 2600 лет — одна из старейших еврейских общин в мире. Они прослеживают своё прибытие до вавилонского пленения. Тёплый кирпичный фасад и скромные размеры синагоги скрывают её значимость. Грузия — одна из редких стран, где антисемитизм никогда не укоренялся. Говорят, когда евреи прибыли в древнюю Колхиду, им просто сказали найти место и обосноваться. Они так и сделали — и остались.",
				LayerType: "human_story", Duration: 51,
			}},
		},
		// --- 18. Rike Park ---
		{
			Name: "Rike Park", NameRu: "Парк Рике",
			Lat: 41.6926, Lng: 44.8117, Type: "park",
			Address: "Rike Park, Tbilisi", InterestScore: 58,
			StoriesEN: []seedStory{{
				Text:      "Rike Park opened in 2010 as part of Tbilisi's modernization wave. It sits in a bend of the Mtkvari River, offering views of Metekhi Church, Narikala Fortress, and the Presidential Palace. The park's most visible features are two massive metal pipes — designed as a concert hall and exhibition space — that were never completed and now stand as monuments to abandoned ambitions. A free cable car connects the park to Narikala Fortress, making it one of the best starting points for exploring Old Tbilisi.",
				LayerType: "general", Duration: 44,
			}},
			StoriesRU: []seedStory{{
				Text:      "Парк Рике был открыт в 2010 году как часть волны модернизации Тбилиси. Он расположен в излучине реки Мтквари, откуда открываются виды на Метехскую церковь, крепость Нарикала и Президентский дворец. Самые заметные объекты парка — две массивные металлические трубы, задуманные как концертный зал и выставочное пространство — так и не были достроены и теперь стоят как памятники брошенным амбициям. Бесплатная канатная дорога соединяет парк с крепостью Нарикала, делая его одной из лучших отправных точек для исследования Старого Тбилиси.",
				LayerType: "general", Duration: 48,
			}},
		},
		// --- 19. Mother of Georgia (Kartlis Deda) ---
		{
			Name: "Mother of Georgia (Kartlis Deda)", NameRu: "Мать Грузия (Картлис Деда)",
			Lat: 41.6882, Lng: 44.8078, Type: "monument",
			Address: "Sololaki Hill, Tbilisi", InterestScore: 68,
			StoriesEN: []seedStory{{
				Text:      "The 20-meter aluminum statue of Kartlis Deda — Mother of Georgia — stands on Sololaki Hill overlooking the city. Erected in 1958 for Tbilisi's 1,500th anniversary, she holds a wine cup in one hand to greet friends and a sword in the other to fend off enemies. The statue perfectly captures the Georgian character: warmth and hospitality balanced with fierce independence. The original wooden statue by Elguja Amashukeli was replaced by aluminum in 1963, and she has watched silently over the city ever since.",
				LayerType: "general", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Двадцатиметровая алюминиевая статуя Картлис Деда — Мать Грузии — стоит на холме Сололаки, возвышаясь над городом. Установленная в 1958 году к 1500-летию Тбилиси, она держит в одной руке чашу вина для встречи друзей, а в другой — меч для отпора врагам. Статуя идеально передаёт грузинский характер: тепло и гостеприимство в сочетании с яростной независимостью. Оригинальная деревянная статуя работы Элгуджи Амашукели была заменена алюминиевой в 1963 году, и с тех пор она молча наблюдает за городом.",
				LayerType: "general", Duration: 49,
			}},
		},
		// --- 20. Tbilisi Opera and Ballet Theater ---
		{
			Name: "Tbilisi Opera and Ballet Theater", NameRu: "Тбилисский театр оперы и балета",
			Lat: 41.7003, Lng: 44.7948, Type: "building",
			Address: "25 Rustaveli Ave, Tbilisi", InterestScore: 65,
			StoriesEN: []seedStory{{
				Text:      "The Tbilisi Opera and Ballet Theater has stood on Rustaveli Avenue since 1851, making it one of the oldest opera houses in the former Russian Empire. The Moorish Revival building you see today was rebuilt after a fire in 1874. The legendary Georgian opera singer Vano Sarajishvili performed here, and Tchaikovsky himself conducted in this hall. During the Soviet era, the theater maintained its prestige, and even today a ballet ticket costs a fraction of Western prices — a rare case where the Soviet legacy benefits modern audiences.",
				LayerType: "time_shift", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Тбилисский театр оперы и балета стоит на проспекте Руставели с 1851 года, что делает его одним из старейших оперных театров бывшей Российской империи. Здание в мавританском стиле, которое вы видите сегодня, было перестроено после пожара 1874 года. Здесь выступал легендарный грузинский оперный певец Вано Сараджишвили, а сам Чайковский дирижировал в этом зале. В советское время театр сохранил свой престиж, и даже сегодня билет на балет стоит в разы дешевле, чем на Западе — редкий случай, когда советское наследие приносит пользу современному зрителю.",
				LayerType: "time_shift", Duration: 52,
			}},
		},
		// --- 21. Dry Bridge Flea Market ---
		{
			Name: "Dry Bridge Flea Market", NameRu: "Блошиный рынок на Сухом мосту",
			Lat: 41.6966, Lng: 44.8028, Type: "bridge",
			Address: "Dry Bridge, Tbilisi", InterestScore: 63,
			StoriesEN: []seedStory{{
				Text:      "Every day, vendors lay out Soviet-era memorabilia, antique jewelry, old paintings, vinyl records, and curious trinkets on the pavement of Dry Bridge. The market boomed in the 1990s when the Soviet collapse left families selling heirlooms to survive. The bridge itself, built in the 1850s, once crossed the Mtkvari but now spans a park after the river changed course. Haggling is expected. Among the Soviet medals and cracked china, patient searchers sometimes find genuine antiques — Georgian silver, Caucasian daggers, or icons from village churches.",
				LayerType: "human_story", Duration: 50,
			}},
			StoriesRU: []seedStory{{
				Text:      "Каждый день продавцы раскладывают советскую атрибутику, антикварные украшения, старые картины, виниловые пластинки и любопытные безделушки на мостовой Сухого моста. Рынок расцвёл в 1990-х, когда крах Советского Союза заставил семьи продавать фамильные ценности, чтобы выжить. Сам мост, построенный в 1850-х, когда-то пересекал Мтквари, но теперь проходит над парком после того, как река изменила русло. Торг уместен. Среди советских медалей и треснувшего фарфора терпеливые искатели иногда находят настоящий антиквариат — грузинское серебро, кавказские кинжалы или иконы из деревенских церквей.",
				LayerType: "human_story", Duration: 53,
			}},
		},
		// --- 22. Rustaveli Metro Station ---
		{
			Name: "Rustaveli Metro Station", NameRu: "Станция метро «Руставели»",
			Lat: 41.7006, Lng: 44.7951, Type: "building",
			Address: "Rustaveli Ave, Tbilisi", InterestScore: 55,
			StoriesEN: []seedStory{{
				Text:      "Descend the escalators of Rustaveli station and you step into a Soviet-era time capsule. Opened in 1966, Tbilisi's metro was the fourth built in the Soviet Union after Moscow, St. Petersburg, and Kyiv. The stations are deep — some over 60 meters underground — originally designed to double as bomb shelters. Rustaveli station features marble walls and bronze chandeliers typical of the era's belief that public transit should feel palatial. The flat fare of 50 tetri takes you anywhere in the system.",
				LayerType: "hidden_detail", Duration: 44,
			}},
			StoriesRU: []seedStory{{
				Text:      "Спуститесь по эскалаторам станции «Руставели» — и вы окажетесь в советской капсуле времени. Открытое в 1966 году, тбилисское метро стало четвёртым в Советском Союзе после Москвы, Санкт-Петербурга и Киева. Станции глубокие — некоторые более 60 метров под землёй — изначально спроектированные как бомбоубежища. Станция «Руставели» украшена мраморными стенами и бронзовыми люстрами, типичными для эпохи, когда считалось, что общественный транспорт должен выглядеть дворцом. Единый тариф в 50 тетри довезёт вас в любую точку системы.",
				LayerType: "hidden_detail", Duration: 47,
			}},
		},
		// --- 23. Shardeni Street ---
		{
			Name: "Shardeni Street", NameRu: "Улица Шардени",
			Lat: 41.6914, Lng: 44.8073, Type: "street",
			Address: "Shardeni St, Old Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "Shardeni Street is named after Jean Chardin, a 17th-century French jeweler and traveler who spent years in Georgia and left some of the most detailed accounts of life in the Caucasus. Today this narrow pedestrian lane in Old Tbilisi is lined with cafes, wine bars, and art galleries. On warm evenings, tables spill onto the cobblestones and live music drifts from open doorways. The street captures the new Tbilisi — a city that has learned to mix European cafe culture with its own deep-rooted traditions of hospitality.",
				LayerType: "atmosphere", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Улица Шардени названа в честь Жана Шардена, французского ювелира и путешественника XVII века, который провёл годы в Грузии и оставил одни из самых детальных описаний жизни на Кавказе. Сегодня эта узкая пешеходная улочка в Старом Тбилиси уставлена кафе, винными барами и художественными галереями. В тёплые вечера столики выставляют на брусчатку, и живая музыка доносится из открытых дверей. Улица отражает новый Тбилиси — город, научившийся сочетать европейскую кафе-культуру со своими глубоко укоренёнными традициями гостеприимства.",
				LayerType: "atmosphere", Duration: 49,
			}},
		},
		// --- 24. Tbilisi Funicular ---
		{
			Name: "Tbilisi Funicular", NameRu: "Тбилисский фуникулёр",
			Lat: 41.6962, Lng: 44.7909, Type: "building",
			Address: "Chonkadze St, Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "The Tbilisi funicular has been carrying passengers up Mtatsminda Mountain since 1905, when it was built by a Belgian company. The two-station route rises over 300 meters along a steep track through thick forest. At the midway station sits the Funicular Restaurant, once one of the most fashionable dining spots in the Soviet South. Restored after years of disrepair, the funicular reopened in 2012 and now offers one of the best rides in the city — a slow ascent through the trees with the panorama of Tbilisi gradually unfolding behind you.",
				LayerType: "hidden_detail", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Тбилисский фуникулёр перевозит пассажиров на гору Мтацминда с 1905 года, когда был построен бельгийской компанией. Маршрут с двумя станциями поднимается более чем на 300 метров по крутому пути через густой лес. На промежуточной станции расположен ресторан «Фуникулёр», когда-то одно из самых модных мест для обеда на Советском Юге. Восстановленный после лет запустения, фуникулёр был вновь открыт в 2012 году и теперь предлагает одну из лучших поездок в городе — медленный подъём сквозь деревья с панорамой Тбилиси, постепенно разворачивающейся позади.",
				LayerType: "hidden_detail", Duration: 50,
			}},
		},
		// --- 25. Tbilisi Reservoir (Turtle Lake) ---
		{
			Name: "Turtle Lake", NameRu: "Черепашье озеро",
			Lat: 41.7119, Lng: 44.7713, Type: "park",
			Address: "Turtle Lake, Tbilisi", InterestScore: 55,
			StoriesEN: []seedStory{{
				Text:      "Turtle Lake sits in the hills above Tbilisi at about 700 meters elevation, a small natural lake named for the turtles that once populated its shores. On summer weekends, the lake becomes the city's escape — families picnic on the grassy banks, joggers circle the shoreline path, and kayaks glide across the green water. A cable car from Vake Park provides the scenic route up. The lake area also holds an open-air ethnographic museum showcasing traditional houses from every region of Georgia.",
				LayerType: "atmosphere", Duration: 42,
			}},
			StoriesRU: []seedStory{{
				Text:      "Черепашье озеро расположено на холмах над Тбилиси на высоте около 700 метров — небольшое природное озеро, названное в честь черепах, которые когда-то населяли его берега. В летние выходные озеро становится местом отдыха горожан — семьи устраивают пикники на травянистых берегах, бегуны обходят прибрежную тропу, а каяки скользят по зелёной воде. Канатная дорога из парка Ваке обеспечивает живописный подъём. В районе озера также находится этнографический музей под открытым небом с традиционными домами из каждого региона Грузии.",
				LayerType: "atmosphere", Duration: 46,
			}},
		},
		// --- 26. Vake Park ---
		{
			Name: "Vake Park", NameRu: "Парк Ваке",
			Lat: 41.7083, Lng: 44.7735, Type: "park",
			Address: "Vake Park, Tbilisi", InterestScore: 55,
			StoriesEN: []seedStory{{
				Text:      "Vake Park is the lung of Tbilisi's affluent Vake district, a vast green space where joggers, dog walkers, and chess players share the shade of old oaks and pines. At the park's entrance stands a solemn World War II memorial — a towering monument to the 300,000 Georgians who died fighting for the Soviet Union, out of a population of just 3.5 million. The park climbs uphill toward Turtle Lake, and a restored Soviet-era cable car connects the two — a pleasant ride above the treetops.",
				LayerType: "time_shift", Duration: 44,
			}},
			StoriesRU: []seedStory{{
				Text:      "Парк Ваке — лёгкие зажиточного района Ваке, обширное зелёное пространство, где бегуны, собачники и шахматисты делят тень старых дубов и сосен. У входа в парк стоит торжественный мемориал Второй мировой войны — возвышающийся монумент 300 000 грузин, погибших в боях за Советский Союз при населении всего 3,5 миллиона. Парк поднимается к Черепашьему озеру, и восстановленная советская канатная дорога соединяет их — приятная поездка над верхушками деревьев.",
				LayerType: "time_shift", Duration: 47,
			}},
		},
		// --- 27. Georgian National Gallery (Blue Gallery) ---
		{
			Name: "Georgian National Gallery", NameRu: "Национальная галерея Грузии",
			Lat: 41.7010, Lng: 44.7950, Type: "museum",
			Address: "3 Rustaveli Ave (2nd floor), Tbilisi", InterestScore: 62,
			StoriesEN: []seedStory{{
				Text:      "On the upper floors of the Georgian National Museum building sits the National Gallery, home to the finest collection of Georgian art. The star exhibit is the Treasury — a darkened room filled with medieval Georgian goldwork, cloisonne enamel icons, and jeweled crosses dating back to the 8th century. The craftsmanship rivals Byzantine masters. One icon, the Khakhuli Triptych from the 12th century, is considered one of the greatest examples of Georgian enamel art. The gallery reveals a sophisticated artistic tradition that most visitors never knew existed.",
				LayerType: "hidden_detail", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "На верхних этажах здания Национального музея находится Национальная галерея, хранящая лучшую коллекцию грузинского искусства. Главный экспонат — Сокровищница, затемнённый зал, наполненный средневековым грузинским золотом, перегородчатыми эмалевыми иконами и украшенными камнями крестами, датируемыми VIII веком. Мастерство не уступает византийским мастерам. Одна из икон, Хахульский триптих XII века, считается одним из величайших образцов грузинского эмалевого искусства. Галерея раскрывает изощрённую художественную традицию, о существовании которой большинство посетителей даже не подозревали.",
				LayerType: "hidden_detail", Duration: 50,
			}},
		},
		// --- 28. Chronicle of Georgia ---
		{
			Name: "Chronicle of Georgia", NameRu: "Хроника Грузии",
			Lat: 41.7471, Lng: 44.7569, Type: "monument",
			Address: "Tbilisi Sea area, Tbilisi", InterestScore: 64,
			StoriesEN: []seedStory{{
				Text:      "On a hill overlooking the Tbilisi Sea stands a massive monument that few tourists find. The Chronicle of Georgia, designed by sculptor Zurab Tsereteli, consists of 16 stone pillars, each 35 meters tall, carved with scenes from Georgian history and biblical events. Construction began in 1985 and was never fully finished — the base and interior chapel remain incomplete. Yet this gives the monument a powerful unfinished quality, as if Georgia's story is still being carved. At sunset, the pillars glow golden against the reservoir below.",
				LayerType: "atmosphere", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "На холме с видом на Тбилисское море стоит массивный монумент, который находят немногие туристы. Хроника Грузии, созданная скульптором Зурабом Церетели, состоит из 16 каменных столпов высотой 35 метров, покрытых сценами из истории Грузии и библейскими сюжетами. Строительство началось в 1985 году и так и не было завершено — основание и внутренняя часовня остаются незаконченными. Однако это придаёт монументу мощное ощущение незавершённости, словно история Грузии ещё высекается. На закате столпы сияют золотом на фоне водохранилища внизу.",
				LayerType: "atmosphere", Duration: 49,
			}},
		},
		// --- 29. Mtatsminda Pantheon ---
		{
			Name: "Mtatsminda Pantheon", NameRu: "Пантеон Мтацминда",
			Lat: 41.6929, Lng: 44.7881, Type: "monument",
			Address: "Mtatsminda, Tbilisi", InterestScore: 63,
			StoriesEN: []seedStory{{
				Text:      "Halfway up Mtatsminda Mountain, surrounding the small church of St. David, lies the Pantheon — Georgia's most sacred burial ground. Here rest the giants of Georgian culture: the poet Ilia Chavchavadze, the writer Akaki Tsereteli, the artist Niko Pirosmani, and many others. The most famous grave belongs to Alexandre Griboyedov, the Russian diplomat and playwright, buried here beside his Georgian wife Princess Nino Chavchavadze. Her epitaph reads: 'Your mind and deeds are immortal in Russian memory, but why did my love outlive you?'",
				LayerType: "human_story", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "На полпути к вершине горы Мтацминда, вокруг небольшой церкви Святого Давида, расположен Пантеон — самое священное место захоронения Грузии. Здесь покоятся гиганты грузинской культуры: поэт Илья Чавчавадзе, писатель Акакий Церетели, художник Нико Пиросмани и многие другие. Самая знаменитая могила принадлежит Александру Грибоедову, русскому дипломату и драматургу, похороненному здесь рядом с женой — грузинской княжной Нино Чавчавадзе. Её эпитафия гласит: «Ум и дела твои бессмертны в памяти русской, но для чего пережила тебя любовь моя?»",
				LayerType: "human_story", Duration: 52,
			}},
		},
		// --- 30. Presidential Palace ---
		{
			Name: "Presidential Palace", NameRu: "Президентский дворец",
			Lat: 41.6921, Lng: 44.8136, Type: "building",
			Address: "Avlabari, Tbilisi", InterestScore: 58,
			StoriesEN: []seedStory{{
				Text:      "The Presidential Palace of Georgia sits on a hill in Avlabari, its glass dome visible from many parts of the city. Built during Saakashvili's presidency and completed in 2009, the palace was controversial from the start — critics called it an extravagant vanity project in a country with pressing needs. The building blends neoclassical columns with a massive egg-shaped glass cupola. When the capital functions shifted to a new parliament in Kutaisi in 2012, the palace became somewhat quieter, though it remains the official presidential residence.",
				LayerType: "general", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Президентский дворец Грузии расположен на холме в Авлабари, его стеклянный купол виден из многих частей города. Построенный при президентстве Саакашвили и завершённый в 2009 году, дворец был спорным с самого начала — критики называли его расточительным проектом тщеславия в стране с насущными проблемами. Здание сочетает неоклассические колонны с массивным яйцевидным стеклянным куполом. Когда столичные функции были перенесены в новый парламент в Кутаиси в 2012 году, дворец стал несколько тише, хотя остаётся официальной президентской резиденцией.",
				LayerType: "general", Duration: 50,
			}},
		},
		// --- 31. Tbilisi Cable Car ---
		{
			Name: "Tbilisi Cable Car", NameRu: "Тбилисская канатная дорога",
			Lat: 41.6920, Lng: 44.8108, Type: "building",
			Address: "Rike Park to Narikala, Tbilisi", InterestScore: 62,
			StoriesEN: []seedStory{{
				Text:      "The Tbilisi cable car glides silently from Rike Park up to Narikala Fortress, offering three minutes of aerial views over the Old Town. Opened in 2012, the modern gondolas replace what was once a harrowing Soviet cable car with open benches. The ride takes you directly over the Mtkvari River and rooftops of the oldest neighborhoods. At the top, you step out onto Sololaki Hill, where the fortress walls and the statue of Mother Georgia greet you. The return ride at sunset is among the finest experiences Tbilisi offers.",
				LayerType: "atmosphere", Duration: 42,
			}},
			StoriesRU: []seedStory{{
				Text:      "Тбилисская канатная дорога бесшумно скользит от парка Рике к крепости Нарикала, предлагая три минуты воздушных видов над Старым городом. Открытая в 2012 году, современные кабины заменили некогда жуткую советскую канатку с открытыми скамейками. Поездка проходит прямо над рекой Мтквари и крышами старейших кварталов. Наверху вы выходите на холм Сололаки, где вас встречают крепостные стены и статуя Матери Грузии. Обратная поездка на закате — одно из лучших впечатлений, которые предлагает Тбилиси.",
				LayerType: "atmosphere", Duration: 45,
			}},
		},
		// --- 32. Gabriadze Puppet Theater ---
		{
			Name: "Gabriadze Puppet Theater", NameRu: "Театр марионеток Габриадзе",
			Lat: 41.6919, Lng: 44.8066, Type: "building",
			Address: "Shavteli St 26, Old Tbilisi", InterestScore: 64,
			StoriesEN: []seedStory{{
				Text:      "Rezo Gabriadze's Puppet Theater is a tiny venue on Shavteli Street that punches far above its size. Founded in 1981, it stages performances that move adults to tears with hand-crafted puppets and stories that blend Georgian folklore with universal themes of love, loss, and absurdity. Gabriadze, who is also a painter, sculptor, and film director, designed every element of the theater himself — from the chairs to the puppets to the quirky facade. Performances sell out months in advance. The attached cafe serves the best coffee in Old Tbilisi.",
				LayerType: "human_story", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Театр марионеток Резо Габриадзе — крошечная площадка на улице Шавтели, которая далеко превосходит свои размеры. Основанный в 1981 году, он ставит спектакли, которые доводят взрослых до слёз с помощью кукол ручной работы и историй, смешивающих грузинский фольклор с универсальными темами любви, потери и абсурда. Габриадзе, который также является художником, скульптором и кинорежиссёром, спроектировал каждый элемент театра сам — от стульев до кукол и причудливого фасада. Билеты раскупаются за месяцы. Прилегающее кафе подаёт лучший кофе в Старом Тбилиси.",
				LayerType: "human_story", Duration: 50,
			}},
		},
		// --- 33. Betlemi Street & Quarter ---
		{
			Name: "Betlemi Quarter", NameRu: "Квартал Бетлеми",
			Lat: 41.6888, Lng: 44.8073, Type: "district",
			Address: "Betlemi St, Old Tbilisi", InterestScore: 62,
			StoriesEN: []seedStory{{
				Text:      "Betlemi Quarter, named after a Bethlehem church that once stood here, is one of the most atmospheric corners of Old Tbilisi. Narrow streets wind uphill past painted wooden balconies, crumbling brick walls, and hidden courtyards where grapevines cling to iron trellises. The quarter has been slowly restored, and galleries, guesthouses, and workshops now occupy buildings that were nearly abandoned a decade ago. If you look carefully at the doorframes, you can find carvings in Georgian, Armenian, and Persian scripts — evidence of the families who shared these streets for centuries.",
				LayerType: "hidden_detail", Duration: 50,
			}},
			StoriesRU: []seedStory{{
				Text:      "Квартал Бетлеми, названный в честь когда-то стоявшей здесь Вифлеемской церкви, — один из самых атмосферных уголков Старого Тбилиси. Узкие улочки вьются вверх мимо расписных деревянных балконов, осыпающихся кирпичных стен и скрытых дворов, где виноградные лозы цепляются за железные шпалеры. Квартал постепенно реставрируется, и галереи, гостевые дома и мастерские теперь занимают здания, почти заброшенные десять лет назад. Если присмотреться к дверным рамам, можно найти резьбу на грузинском, армянском и персидском письме — свидетельство семей, деливших эти улицы веками.",
				LayerType: "hidden_detail", Duration: 53,
			}},
		},
		// --- 34. Tbilisi Open Air Museum of Ethnography ---
		{
			Name: "Ethnographic Museum", NameRu: "Этнографический музей",
			Lat: 41.7160, Lng: 44.7703, Type: "museum",
			Address: "Turtle Lake Road, Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "Spread across a wooded hillside near Turtle Lake, the Open Air Ethnographic Museum preserves traditional houses from every corner of Georgia. Founded in 1966, it contains over 70 buildings — from the stone towers of Svaneti to the flat-roofed houses of Kakheti to the wooden cottages of Adjara. Each house is furnished with authentic tools, textiles, and utensils. Walking through the museum is like traveling across Georgia in miniature, and the forest setting makes it feel more like a village than a museum. The variety reveals how dramatically Georgia's landscape shaped its regional cultures.",
				LayerType: "general", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Раскинувшийся на лесистом склоне холма у Черепашьего озера, Этнографический музей под открытым небом сохраняет традиционные дома из каждого уголка Грузии. Основанный в 1966 году, он содержит более 70 зданий — от каменных башен Сванетии до домов с плоскими крышами Кахетии и деревянных коттеджей Аджарии. Каждый дом обставлен подлинными инструментами, текстилем и утварью. Прогулка по музею подобна путешествию по Грузии в миниатюре, а лесная обстановка создаёт ощущение деревни, а не музея. Разнообразие раскрывает, насколько драматично ландшафт Грузии сформировал её региональные культуры.",
				LayerType: "general", Duration: 52,
			}},
		},
		// --- 35. Jan Shardeni Street Statue ---
		{
			Name: "Tamada Statue", NameRu: "Статуя Тамады",
			Lat: 41.6912, Lng: 44.8076, Type: "monument",
			Address: "Shardeni St, Old Tbilisi", InterestScore: 55,
			StoriesEN: []seedStory{{
				Text:      "The small bronze statue on Shardeni Street depicts a tamada — the toastmaster of the Georgian feast, or supra. He raises a drinking horn high in an eternal toast. The statue is a modern replica of a 7th-century BC bronze figurine found in Vani, western Georgia, proving that the Georgian tradition of elaborate feasting is at least 2,700 years old. The tamada is one of the most important roles in Georgian culture — he leads the evening through a structured series of toasts to God, to Georgia, to the dead, to love, and to life. Missing a supra means missing the soul of Georgia.",
				LayerType: "human_story", Duration: 50,
			}},
			StoriesRU: []seedStory{{
				Text:      "Маленькая бронзовая статуя на улице Шардени изображает тамаду — распорядителя грузинского застолья, или супры. Он высоко поднимает рог в вечном тосте. Статуя — современная копия бронзовой фигурки VII века до нашей эры, найденной в Вани, западная Грузия, что доказывает: грузинская традиция пышных застолий насчитывает не менее 2700 лет. Тамада — одна из важнейших ролей в грузинской культуре: он ведёт вечер через структурированную серию тостов за Бога, за Грузию, за умерших, за любовь и за жизнь. Пропустить супру — значит пропустить душу Грузии.",
				LayerType: "human_story", Duration: 53,
			}},
		},
		// --- 36. Tbilisi Circus ---
		{
			Name: "Tbilisi State Circus", NameRu: "Тбилисский государственный цирк",
			Lat: 41.7116, Lng: 44.7792, Type: "building",
			Address: "Heroes Square, Tbilisi", InterestScore: 50,
			StoriesEN: []seedStory{{
				Text:      "The Tbilisi State Circus building near Heroes Square is a striking Soviet modernist structure completed in 1940. Its round arena and distinctive dome were once filled with acrobats, clowns, and animal acts that delighted generations of Georgian children. During the Soviet era, the circus was one of Tbilisi's premier entertainment venues. After years of decline and partial closure, the building itself has become a landmark — a reminder of an era when the state invested heavily in public entertainment and the circus was considered a serious art form.",
				LayerType: "time_shift", Duration: 42,
			}},
			StoriesRU: []seedStory{{
				Text:      "Здание Тбилисского государственного цирка у площади Героев — выразительное сооружение советского модернизма, завершённое в 1940 году. Его круглая арена и характерный купол когда-то были наполнены акробатами, клоунами и номерами с животными, восхищавшими поколения грузинских детей. В советское время цирк был одним из главных развлекательных мест Тбилиси. После лет упадка и частичного закрытия само здание стало достопримечательностью — напоминанием об эпохе, когда государство щедро инвестировало в общественное развлечение, а цирк считался серьёзным искусством.",
				LayerType: "time_shift", Duration: 46,
			}},
		},
		// --- 37. Vera District ---
		{
			Name: "Vera District", NameRu: "Район Вера",
			Lat: 41.7069, Lng: 44.7852, Type: "district",
			Address: "Vera, Tbilisi", InterestScore: 55,
			StoriesEN: []seedStory{{
				Text:      "Vera is one of Tbilisi's most charming residential neighborhoods, a grid of tree-lined streets filled with late 19th-century houses and small parks. Unlike the tourist-heavy Old Town, Vera feels like the city Tbilisians actually live in. The neighborhood grew up around the Vera Garden — now called Pushkin Park — and retains a village-like atmosphere despite being minutes from Rustaveli Avenue. Walking through Vera, you notice details missed in busier districts: carved wooden doors, tiled stoops, garden walls heavy with ivy, and the sound of piano practice drifting from an open window.",
				LayerType: "atmosphere", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Вера — один из самых очаровательных жилых районов Тбилиси, сетка обсаженных деревьями улиц с домами конца XIX века и маленькими парками. В отличие от переполненного туристами Старого города, Вера ощущается как город, в котором действительно живут тбилисцы. Район вырос вокруг Верийского сада — ныне Пушкинского парка — и сохраняет деревенскую атмосферу, несмотря на близость к проспекту Руставели. Гуляя по Вере, замечаешь детали, которые теряются в более оживлённых районах: резные деревянные двери, выложенные плиткой крыльца, садовые стены, увитые плющом, и звуки фортепьянных упражнений из открытого окна.",
				LayerType: "atmosphere", Duration: 50,
			}},
		},
		// --- 38. Orbeliani Baths ---
		{
			Name: "Orbeliani Baths (Chreli Abano)", NameRu: "Бани Орбелиани (Пёстрая баня)",
			Lat: 41.6883, Lng: 44.8100, Type: "building",
			Address: "Abano St 2, Old Tbilisi", InterestScore: 68,
			StoriesEN: []seedStory{{
				Text:      "The Orbeliani Baths, also called Chreli Abano or the Blue Bath, feature the most ornate facade in Abanotubani — a turquoise-tiled front inspired by Iranian mosque architecture. Built in the 17th century by the noble Orbeliani family, this is the same bathhouse where Pushkin soaked in 1829 and wrote his famous praise. The water rises naturally at about 40 degrees Celsius, rich in sulfur and believed to cure skin ailments. Inside, the private rooms are built under the same brick domes you see from street level, each with its own hot pool.",
				LayerType: "hidden_detail", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Бани Орбелиани, также называемые Пёстрой баней, имеют самый нарядный фасад в Абанотубани — бирюзовую мозаику, вдохновлённую архитектурой иранских мечетей. Построенные в XVII веке знатным родом Орбелиани, именно здесь в 1829 году парился Пушкин и написал свой знаменитый восторженный отзыв. Вода поднимается естественным образом при температуре около 40 градусов Цельсия, богатая серой и, как считается, излечивающая кожные заболевания. Внутри частные комнаты построены под теми же кирпичными куполами, которые видны с улицы, каждая с собственным горячим бассейном.",
				LayerType: "hidden_detail", Duration: 50,
			}},
		},
		// --- 39. Tbilisi Marjanishvili Theater ---
		{
			Name: "Marjanishvili Theater", NameRu: "Театр Марджанишвили",
			Lat: 41.7043, Lng: 44.8003, Type: "building",
			Address: "Marjanishvili Square 8, Tbilisi", InterestScore: 55,
			StoriesEN: []seedStory{{
				Text:      "The Marjanishvili Theater, named after the pioneering Georgian director Kote Marjanishvili, sits in the lively square that also bears his name. Founded in 1928, it has long been considered the more experimental of Tbilisi's two main drama theaters. While the Rustaveli Theater represents classical tradition, Marjanishvili pushes boundaries. The theater has won awards at the Edinburgh Festival and toured internationally, bringing Georgian theatrical art to audiences who were surprised to find a small Caucasian country producing world-class drama.",
				LayerType: "general", Duration: 42,
			}},
			StoriesRU: []seedStory{{
				Text:      "Театр Марджанишвили, названный в честь новаторского грузинского режиссёра Котэ Марджанишвили, расположен на оживлённой площади, также носящей его имя. Основанный в 1928 году, он давно считается более экспериментальным из двух главных драматических театров Тбилиси. В то время как Театр Руставели представляет классическую традицию, Марджанишвили раздвигает границы. Театр получал награды на Эдинбургском фестивале и гастролировал по миру, знакомя зрителей с грузинским театральным искусством, которые удивлялись, что маленькая кавказская страна создаёт драматургию мирового класса.",
				LayerType: "general", Duration: 46,
			}},
		},
		// --- 40. Tbilisi Central Market (Dezerter Bazaar) ---
		{
			Name: "Dezerter Bazaar", NameRu: "Дезертирский базар",
			Lat: 41.7138, Lng: 44.8072, Type: "building",
			Address: "Station Square area, Tbilisi", InterestScore: 58,
			StoriesEN: []seedStory{{
				Text:      "The Dezerter Bazaar, near the central railway station, is Tbilisi's largest traditional market. Its name comes from a less proud origin — deserters from the 1921 war against Soviet invasion once gathered here to trade. Today the market is a sensory overload of churchkhela hanging in colorful rows, giant wheels of sulguni cheese, mountains of spices, fresh herbs, and dried fruits. The vendors, mostly women from the regions, will insist you taste everything before buying. The basement level sells meat and fish; the upper floors hold clothing and household goods. This is the real Tbilisi, far from the polished tourist streets.",
				LayerType: "human_story", Duration: 50,
			}},
			StoriesRU: []seedStory{{
				Text:      "Дезертирский базар, рядом с центральным железнодорожным вокзалом, — крупнейший традиционный рынок Тбилиси. Название происходит от менее гордого прошлого — дезертиры из войны 1921 года против советского вторжения когда-то собирались здесь для торговли. Сегодня рынок — это сенсорная перегрузка: чурчхела, свисающая красочными рядами, гигантские круги сыра сулугуни, горы специй, свежая зелень и сухофрукты. Продавцы, в основном женщины из регионов, настоят, чтобы вы попробовали всё перед покупкой. В подвале продают мясо и рыбу; верхние этажи — одежда и хозтовары. Это настоящий Тбилиси, далёкий от полированных туристических улиц.",
				LayerType: "human_story", Duration: 54,
			}},
		},
		// --- 41. Tbilisi Central Railway Station ---
		{
			Name: "Tbilisi Central Railway Station", NameRu: "Центральный железнодорожный вокзал Тбилиси",
			Lat: 41.7143, Lng: 44.8044, Type: "building",
			Address: "Station Square 1, Tbilisi", InterestScore: 52,
			StoriesEN: []seedStory{{
				Text:      "Tbilisi's central railway station was originally built in the 1870s when the Transcaucasian Railway connected Tbilisi to the Black Sea port of Poti. The current building dates from a Soviet-era reconstruction. The station was once a grand gateway to the empire — trains left for Moscow, Baku, and Yerevan. Today rail connections are fewer but the night train to Batumi remains a beloved Georgian experience. Station Square outside is the city's busiest transport hub, where marshrutka minibuses depart for every corner of the country.",
				LayerType: "time_shift", Duration: 44,
			}},
			StoriesRU: []seedStory{{
				Text:      "Центральный железнодорожный вокзал Тбилиси был первоначально построен в 1870-х годах, когда Закавказская железная дорога соединила Тбилиси с черноморским портом Поти. Нынешнее здание датируется советской реконструкцией. Вокзал когда-то был парадными воротами империи — поезда отправлялись в Москву, Баку и Ереван. Сегодня железнодорожных соединений меньше, но ночной поезд в Батуми остаётся любимым грузинским путешествием. Привокзальная площадь — самый оживлённый транспортный узел города, откуда маршрутки отправляются в каждый уголок страны.",
				LayerType: "time_shift", Duration: 48,
			}},
		},
		// --- 42. Kashveti Church ---
		{
			Name: "Kashveti Church", NameRu: "Церковь Кашвети",
			Lat: 41.6985, Lng: 44.7991, Type: "church",
			Address: "Rustaveli Ave 9, Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "Kashveti Church sits directly across Rustaveli Avenue from the old Parliament building, a quiet stone church flanked by the avenue's busiest cafes. Built in 1910, it replaced a much older church on the same site. Its name means 'birth from stone' — according to legend, a monk was falsely accused of making a nun pregnant, and he prayed that she would give birth to a stone to prove his innocence, which she did. Inside, the church holds frescoes by the painter Lado Gudiashvili, whose modernist style caused controversy when they were unveiled in the 1940s.",
				LayerType: "hidden_detail", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Церковь Кашвети стоит прямо напротив старого здания Парламента на проспекте Руставели — тихая каменная церковь в окружении самых оживлённых кафе проспекта. Построенная в 1910 году, она заменила гораздо более старую церковь на этом месте. Название означает «рождение из камня» — по легенде, монаха ложно обвинили в том, что он сделал монахиню беременной, и он молился, чтобы она родила камень в доказательство его невиновности, что и произошло. Внутри церковь хранит фрески художника Ладо Гудиашвили, чей модернистский стиль вызвал полемику при их открытии в 1940-х годах.",
				LayerType: "hidden_detail", Duration: 52,
			}},
		},
		// --- 43. Sololaki District ---
		{
			Name: "Sololaki District", NameRu: "Район Сололаки",
			Lat: 41.6920, Lng: 44.8012, Type: "district",
			Address: "Sololaki, Tbilisi", InterestScore: 62,
			StoriesEN: []seedStory{{
				Text:      "Sololaki is the elegant residential district that climbs from Rustaveli Avenue toward the ridge where Narikala Fortress stands. In the late 19th century, it was where Tbilisi's wealthy merchant families built their homes — Georgians, Armenians, Persians, and Germans side by side. The architecture is an eclectic mix of Art Nouveau facades, carved wooden balconies, and ornate iron gates. Many buildings show signs of loving restoration alongside others in beautiful decay. Walking Sololaki's steep, quiet streets is like reading the chapters of Tbilisi's multicultural past in stone and wood.",
				LayerType: "atmosphere", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Сололаки — элегантный жилой район, поднимающийся от проспекта Руставели к хребту, где стоит крепость Нарикала. В конце XIX века здесь строили дома состоятельные купеческие семьи Тбилиси — грузины, армяне, персы и немцы бок о бок. Архитектура — эклектичная смесь фасадов в стиле модерн, резных деревянных балконов и витиеватых железных ворот. Многие здания демонстрируют следы любовной реставрации наряду с другими в красивом упадке. Прогулка по крутым тихим улицам Сололаки подобна чтению глав мультикультурного прошлого Тбилиси в камне и дереве.",
				LayerType: "atmosphere", Duration: 50,
			}},
		},
		// --- 44. Narikala Cable Car Upper Station (viewpoint) ---
		{
			Name: "Old Town Viewpoint (Narikala)", NameRu: "Смотровая площадка Старого города (Нарикала)",
			Lat: 41.6878, Lng: 44.8083, Type: "monument",
			Address: "Narikala Ridge, Old Tbilisi", InterestScore: 72,
			StoriesEN: []seedStory{{
				Text:      "Stand here on the ridge beside Narikala Fortress and you see why Tbilisi was built in this spot. Below, the Mtkvari River carves a tight gorge through the hills, and the city fills every available space — climbing the slopes, squeezing along the riverbanks, spilling over ridgelines. You can pick out the sulfur baths by their brick domes, the glass Bridge of Peace catching the light, the Sameba Cathedral's golden dome on the opposite hill, and the sprawl of Soviet apartment blocks beyond. This single viewpoint contains 1,500 years of urban history laid out like a map.",
				LayerType: "atmosphere", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Встаньте здесь, на хребте рядом с крепостью Нарикала, и вы поймёте, почему Тбилиси был основан именно здесь. Внизу река Мтквари прорезает узкое ущелье сквозь холмы, а город заполняет каждый доступный клочок — карабкается по склонам, теснится вдоль берегов, перетекает через гребни. Можно различить серные бани по их кирпичным куполам, стеклянный Мост Мира, ловящий свет, золотой купол собора Самеба на противоположном холме и простирающиеся советские жилые кварталы за ними. Эта единственная смотровая площадка вмещает 1500 лет городской истории, разложенной как карта.",
				LayerType: "atmosphere", Duration: 52,
			}},
		},
		// --- 45. Statue of King Vakhtang Gorgasali ---
		{
			Name: "King Vakhtang Gorgasali Statue", NameRu: "Памятник царю Вахтангу Горгасали",
			Lat: 41.6909, Lng: 44.8117, Type: "monument",
			Address: "Metekhi, Tbilisi", InterestScore: 65,
			StoriesEN: []seedStory{{
				Text:      "The equestrian statue of King Vakhtang Gorgasali towers over the Metekhi cliff, one of the most iconic silhouettes in Tbilisi's skyline. Vakhtang, who ruled in the 5th century, is credited with founding the city after discovering the hot springs of Abanotubani. The 'wolf-headed' king — his nickname comes from his helmet — moved Georgia's capital from Mtskheta to Tbilisi and fought to keep the kingdom independent from both Persia and Byzantium. The statue, erected in 1967, shows him gazing across the river as if still keeping watch over the city he created.",
				LayerType: "human_story", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Конная статуя царя Вахтанга Горгасали возвышается над Метехской скалой — один из самых знаковых силуэтов в панораме Тбилиси. Вахтанг, правивший в V веке, считается основателем города после открытия горячих источников Абанотубани. «Волкоголовый» царь — прозвище происходит от его шлема — перенёс столицу Грузии из Мцхеты в Тбилиси и боролся за независимость царства и от Персии, и от Византии. Статуя, установленная в 1967 году, изображает его смотрящим через реку, словно он до сих пор охраняет созданный им город.",
				LayerType: "human_story", Duration: 50,
			}},
		},
		// --- 46. Avlabari District ---
		{
			Name: "Avlabari District", NameRu: "Район Авлабари",
			Lat: 41.6942, Lng: 44.8150, Type: "district",
			Address: "Avlabari, Tbilisi", InterestScore: 56,
			StoriesEN: []seedStory{{
				Text:      "Avlabari sits on the left bank of the Mtkvari, historically the Armenian quarter of Tbilisi. For centuries, Armenians made up a significant portion of the city's population, and their churches, schools, and businesses dominated this hillside neighborhood. The district was also home to the underground printing press where young revolutionaries, including Stalin, produced illegal socialist newspapers in the early 1900s. Today Avlabari is overshadowed by the massive Sameba Cathedral at its summit, but the winding streets below still hold traces of its layered, multinational past.",
				LayerType: "time_shift", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Авлабари расположен на левом берегу Мтквари, исторически армянский квартал Тбилиси. Веками армяне составляли значительную часть населения города, и их церкви, школы и предприятия доминировали на этом склоне холма. Район также был домом подпольной типографии, где молодые революционеры, включая Сталина, печатали нелегальные социалистические газеты в начале 1900-х. Сегодня Авлабари затенён массивным собором Самеба на своей вершине, но извилистые улочки внизу всё ещё хранят следы своего многослойного многонационального прошлого.",
				LayerType: "time_shift", Duration: 50,
			}},
		},
		// --- 47. Tbilisi Concert Hall ---
		{
			Name: "Tbilisi Concert Hall", NameRu: "Тбилисский концертный зал",
			Lat: 41.7006, Lng: 44.7929, Type: "building",
			Address: "1 Meliton Balanchivadze St, Tbilisi", InterestScore: 53,
			StoriesEN: []seedStory{{
				Text:      "The Tbilisi Concert Hall stands just behind the Rustaveli Theater, a modernist building from the 1970s that hosts the Georgian Philharmonic Orchestra. Georgia has a remarkable musical tradition — polyphonic singing was one of the first cultural practices inscribed on UNESCO's Intangible Heritage list. The concert hall regularly features both classical Western repertoire and distinctly Georgian compositions. Tickets are astonishingly affordable, and the acoustics are excellent. On any given evening, you might hear Beethoven followed by a piece by the Georgian composer Giya Kancheli.",
				LayerType: "general", Duration: 44,
			}},
			StoriesRU: []seedStory{{
				Text:      "Тбилисский концертный зал стоит за Театром Руставели — модернистское здание 1970-х годов, в котором выступает Грузинский филармонический оркестр. У Грузии замечательная музыкальная традиция — полифоническое пение было одной из первых культурных практик, включённых в список нематериального наследия ЮНЕСКО. Концертный зал регулярно представляет и классический западный репертуар, и самобытные грузинские композиции. Билеты удивительно доступны, а акустика превосходна. В любой вечер можно услышать Бетховена, за которым следует произведение грузинского композитора Гии Канчели.",
				LayerType: "general", Duration: 48,
			}},
		},
		// --- 48. Leselidze Street ---
		{
			Name: "Leselidze Street", NameRu: "Улица Леселидзе",
			Lat: 41.6925, Lng: 44.8050, Type: "street",
			Address: "Kote Abkhazi (Leselidze) St, Old Tbilisi", InterestScore: 58,
			StoriesEN: []seedStory{{
				Text:      "Leselidze Street — officially renamed Kote Abkhazi Street — is the main pedestrian artery of Old Tbilisi, running from Freedom Square deep into the historic quarter. The street is lined with souvenir shops, wine bars, and bakeries where you can watch tonis puri being slapped onto the walls of a tone oven. In the evenings, street musicians set up on corners and the narrow lane fills with a mix of tourists and locals. Look up at the upper stories and you see the real Old Tbilisi: carved balconies, hanging laundry, and potted plants precariously balanced on windowsills.",
				LayerType: "atmosphere", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Улица Леселидзе — официально переименованная в улицу Коте Абхази — главная пешеходная артерия Старого Тбилиси, идущая от площади Свободы вглубь исторического квартала. Улица уставлена сувенирными магазинами, винными барами и пекарнями, где можно наблюдать, как тонис пури шлёпают о стенки тоне. По вечерам уличные музыканты располагаются на углах, и узкая улочка наполняется смесью туристов и местных жителей. Посмотрите на верхние этажи — и увидите настоящий Старый Тбилиси: резные балконы, развешанное бельё и горшки с цветами, ненадёжно балансирующие на подоконниках.",
				LayerType: "atmosphere", Duration: 50,
			}},
		},
		// --- 49. Wine Factory N1 ---
		{
			Name: "Wine Factory N1", NameRu: "Винная фабрика №1",
			Lat: 41.6940, Lng: 44.8103, Type: "building",
			Address: "Gorgasali St 6, Old Tbilisi", InterestScore: 58,
			StoriesEN: []seedStory{{
				Text:      "Wine Factory N1 is a 19th-century wine cellar converted into one of Tbilisi's most atmospheric tasting rooms. The stone-walled cellars once stored wine for the Russian Imperial army. Today they house rows of qvevri — the giant clay vessels that Georgia uses for its ancient winemaking method. Georgian qvevri winemaking dates back 8,000 years, making it the oldest wine tradition on Earth. UNESCO recognized it as Intangible Cultural Heritage in 2013. Tasting here, in the cool underground vaults where wine was stored for centuries, connects you to an unbroken chain of human craft.",
				LayerType: "hidden_detail", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Винная фабрика №1 — винный погреб XIX века, превращённый в одну из самых атмосферных дегустационных залов Тбилиси. Каменные подвалы когда-то хранили вино для Российской императорской армии. Сегодня здесь стоят ряды квеври — гигантских глиняных сосудов, которые Грузия использует для своего древнего метода виноделия. Грузинское квеврийное виноделие насчитывает 8000 лет, что делает его старейшей винодельческой традицией на Земле. ЮНЕСКО признала его нематериальным культурным наследием в 2013 году. Дегустация здесь, в прохладных подземных сводах, где вино хранилось веками, связывает вас с непрерывной цепью человеческого мастерства.",
				LayerType: "hidden_detail", Duration: 52,
			}},
		},
		// --- 50. Heroes Square ---
		{
			Name: "Heroes Square", NameRu: "Площадь Героев",
			Lat: 41.7115, Lng: 44.7820, Type: "square",
			Address: "Heroes Square, Tbilisi", InterestScore: 52,
			StoriesEN: []seedStory{{
				Text:      "Heroes Square is a major traffic roundabout in central Tbilisi, dominated by a soaring monument to Georgian soldiers who died in the wars of the 1990s — the conflicts in Abkhazia and South Ossetia that cost Georgia a quarter of its territory. The pillar, topped by the figure of St. George, was erected in 2010. For Georgians, this square is a reminder of painful recent history that still shapes the country's politics and identity. The surrounding area has been redeveloping rapidly, with new hotels and office buildings rising around the older Soviet blocks.",
				LayerType: "time_shift", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Площадь Героев — крупная транспортная развязка в центре Тбилиси, над которой возвышается монумент грузинским солдатам, погибшим в войнах 1990-х — конфликтах в Абхазии и Южной Осетии, стоивших Грузии четверти её территории. Колонна, увенчанная фигурой Святого Георгия, была установлена в 2010 году. Для грузин эта площадь — напоминание о болезненной недавней истории, которая до сих пор формирует политику и идентичность страны. Окружающий район быстро перестраивается — новые отели и офисные здания поднимаются среди старых советских кварталов.",
				LayerType: "time_shift", Duration: 50,
			}},
		},
		// --- 51. Niko Pirosmani Museum ---
		{
			Name: "Niko Pirosmani Museum (on Pirosmani St)", NameRu: "Музей Нико Пиросмани",
			Lat: 41.6895, Lng: 44.8051, Type: "museum",
			Address: "Pirosmani St 29, Old Tbilisi", InterestScore: 58,
			StoriesEN: []seedStory{{
				Text:      "Niko Pirosmani is Georgia's most beloved painter — a self-taught artist who lived in poverty, painting tavern signs and shop banners in exchange for food and wine. His naive style, with dark backgrounds and luminous figures, captures Georgian life at the turn of the 20th century: feasts, animals, village scenes, and portraits of ordinary people. He died destitute in 1918. Decades later, the world recognized his genius. There is a famous Georgian song about him buying a million roses for an actress he loved. His paintings now hang in the finest museums, but this small museum on the street named after him shows where his story began.",
				LayerType: "human_story", Duration: 52,
			}},
			StoriesRU: []seedStory{{
				Text:      "Нико Пиросмани — самый любимый художник Грузии, самоучка, живший в бедности и расписывавший вывески духанов и магазинов в обмен на еду и вино. Его наивный стиль с тёмным фоном и светящимися фигурами запечатлел грузинскую жизнь на рубеже XX века: застолья, животных, деревенские сцены и портреты простых людей. Он умер в нищете в 1918 году. Спустя десятилетия мир признал его гений. Существует знаменитая песня о том, как он купил миллион роз для актрисы, которую любил. Его картины теперь висят в лучших музеях, но этот маленький музей на улице его имени показывает, где началась его история.",
				LayerType: "human_story", Duration: 55,
			}},
		},
		// --- 52. Aghmashenebeli Avenue ---
		{
			Name: "Aghmashenebeli Avenue", NameRu: "Проспект Агмашенебели",
			Lat: 41.7082, Lng: 44.7981, Type: "street",
			Address: "Aghmashenebeli Ave, Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "Aghmashenebeli Avenue, named after the medieval king David the Builder, is Tbilisi's second great boulevard — less touristic than Rustaveli but equally rich in character. A tree-lined pedestrian section features beautifully restored 19th-century facades in pastel colors, with restaurants and cafes at street level. This was historically the commercial district of the city's south side, and its architecture reflects the prosperity of Tbilisi's merchant era. The avenue connects the Marjanishvili area with the central railway station, offering a pleasant hour-long walk through a neighborhood that feels authentically local.",
				LayerType: "atmosphere", Duration: 46,
			}},
			StoriesRU: []seedStory{{
				Text:      "Проспект Агмашенебели, названный в честь средневекового царя Давида Строителя, — второй великий бульвар Тбилиси, менее туристический, чем Руставели, но столь же богатый характером. Пешеходный участок с деревьями украшен прекрасно отреставрированными фасадами XIX века в пастельных тонах, с ресторанами и кафе на уровне улицы. Исторически это был торговый район южной части города, и его архитектура отражает процветание купеческой эпохи Тбилиси. Проспект соединяет район Марджанишвили с центральным вокзалом, предлагая приятную часовую прогулку по району, который ощущается подлинно местным.",
				LayerType: "atmosphere", Duration: 50,
			}},
		},
		// --- 53. Fabrika Hostel (former Soviet sewing factory) ---
		{
			Name: "Fabrika", NameRu: "Фабрика",
			Lat: 41.7065, Lng: 44.7998, Type: "building",
			Address: "8 Egnate Ninoshvili St, Tbilisi", InterestScore: 55,
			StoriesEN: []seedStory{{
				Text:      "Fabrika is a Soviet-era sewing factory transformed into a creative hub that perfectly captures Tbilisi's modern identity. The sprawling industrial building now houses a hostel, co-working spaces, artist studios, cafes, and bars arranged around a central courtyard covered in street art. On warm evenings, the courtyard fills with a mix of digital nomads, local artists, and travelers sharing tables and ideas. The conversion preserved the factory's raw concrete and steel bones, layering new life onto old infrastructure — an approach that feels very Tbilisi, a city that has always adapted rather than demolished.",
				LayerType: "general", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Фабрика — советская швейная фабрика, превращённая в креативный хаб, идеально отражающий современную идентичность Тбилиси. Обширное промышленное здание теперь вмещает хостел, коворкинг, художественные мастерские, кафе и бары, расположенные вокруг центрального двора, покрытого стрит-артом. В тёплые вечера двор наполняется смесью цифровых кочевников, местных художников и путешественников, делящих столы и идеи. Преобразование сохранило сырой бетон и стальной каркас фабрики, наслаивая новую жизнь на старую инфраструктуру — подход, очень свойственный Тбилиси, городу, который всегда приспосабливался, а не сносил.",
				LayerType: "general", Duration: 52,
			}},
		},
		// --- 54. Georgian Parliament (old building) ---
		{
			Name: "Former Parliament Building", NameRu: "Бывшее здание Парламента",
			Lat: 41.6986, Lng: 44.7980, Type: "building",
			Address: "8 Rustaveli Ave, Tbilisi", InterestScore: 60,
			StoriesEN: []seedStory{{
				Text:      "The grand neoclassical building on Rustaveli Avenue served as Georgia's Parliament until 2012. Its steps and forecourt have been the stage for some of the most pivotal moments in modern Georgian history. On April 9, 1989, Soviet soldiers attacked a peaceful demonstration here, killing 21 people — an event that galvanized the Georgian independence movement. In 2003, the Rose Revolution unfolded on these same steps when Saakashvili led protesters into the chamber holding roses. A memorial to the 1989 victims stands nearby. Today the building serves governmental functions, but its political symbolism endures.",
				LayerType: "time_shift", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Величественное неоклассическое здание на проспекте Руставели служило Парламентом Грузии до 2012 года. Его ступени и площадь перед ним были сценой для самых поворотных моментов современной грузинской истории. 9 апреля 1989 года советские солдаты атаковали мирную демонстрацию здесь, убив 21 человека — событие, придавшее мощный импульс движению за независимость Грузии. В 2003 году на этих же ступенях развернулась Революция роз, когда Саакашвили повёл протестующих с розами в зал заседаний. Мемориал жертвам 1989 года стоит неподалёку. Сегодня здание выполняет государственные функции, но его политический символизм остаётся.",
				LayerType: "time_shift", Duration: 52,
			}},
		},
		// --- 55. Tbilisi History Museum (Karvasla) ---
		{
			Name: "Tbilisi History Museum (Karvasla)", NameRu: "Музей истории Тбилиси (Карвасла)",
			Lat: 41.6910, Lng: 44.8068, Type: "museum",
			Address: "Sioni St 8, Old Tbilisi", InterestScore: 58,
			StoriesEN: []seedStory{{
				Text:      "The Tbilisi History Museum occupies a 19th-century caravanserai — a traditional inn where merchants on the Silk Road rested, stored goods, and traded. The stone courtyard with its two-story galleries looks much as it did when camel trains arrived from Persia and Turkey. Inside, the museum traces the city's history from its founding through archeological finds, photographs, and artifacts. One room recreates a traditional Tbilisi courtyard home, complete with a grapevine-covered balcony and a wooden staircase worn smooth by generations. The building itself is as much an exhibit as anything inside it.",
				LayerType: "hidden_detail", Duration: 48,
			}},
			StoriesRU: []seedStory{{
				Text:      "Музей истории Тбилиси занимает караван-сарай XIX века — традиционный постоялый двор, где купцы Шёлкового пути отдыхали, хранили товары и торговали. Каменный двор с двухэтажными галереями выглядит почти так же, как когда сюда прибывали караваны верблюдов из Персии и Турции. Внутри музей прослеживает историю города от основания через археологические находки, фотографии и артефакты. Одна комната воссоздаёт традиционный тбилисский дом с двором, виноградным балконом и деревянной лестницей, отполированной поколениями. Само здание — такой же экспонат, как и всё, что в нём находится.",
				LayerType: "hidden_detail", Duration: 52,
			}},
		},
	}
}
