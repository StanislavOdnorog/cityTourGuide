## Core concept

Aha — now it clicks.

You **still use Google Maps to go somewhere**

…but instead of silence/music, your headphones play a **context-aware story about everything you pass**.

So navigation app = unchanged

Your app = **parallel audio reality layer**

> Maps tells you *where to go*
> 
> 
> Your AI tells you *what you’re inside of*
> 

---

## How it feels to the user

You start walking to a café using Google Maps.

In headphones:

> “You’re entering Sololaki — this used to be the rich merchant district… look at the balconies.”
> 

You keep walking.

> “On your left — that courtyard survived the 1902 earthquake.”
> 

You don’t interact.

You don’t open the app again.

You just walk + listen.

---

## Key principle

**Never interrupt navigation**

No buttons

No stops

No routes

No tours

It reacts to your real life movement.

This makes it usable daily — not just tourism.

---

## What the app actually does (simple version)

Your phone constantly reads:

- GPS position
- walking speed
- direction

Every few seconds:

1. Check nearby interesting objects
2. Choose the *best* one in front of user
3. Tell a short story (15–40 sec)

That’s it.

You’re basically building:

> **Spotify for the physical world**
> 

---

## The magic trick (important)

You do NOT narrate every building.

Rules:

- silence is default
- story only when meaningful
- avoid overload
- pacing depends on walking speed

The app must feel like:

a thoughtful human guide who knows when to talk

---

## Why this works with Google Maps

People already walk with:

- earbuds
- maps open
- phone locked

You add value without behavior change.

No learning curve = adoption.

---

## MVP architecture (very realistic)

You don’t integrate into Google Maps.

You just run alongside it.

**Phone**

- background location tracking
- compass heading
- audio playback

**Backend**

- database of points (OSM + Wikipedia)
- prewritten micro-stories
- simple ranking algorithm

No live heavy AI needed initially.

---

## First version scope (important for success)

Do NOT launch world.

Launch:

> “Tbilisi walking companion”
> 

One city = dense stories = magical.

If first walk feels magical → people share.

---

## Monetization

Tourists:

$4.99 city unlock

Locals:

daily walking companion

subscription $3–5/mo

Hotels, hostels, Airbnbs will recommend automatically.

---

## The real differentiation

Audio guides today = planned activity

Yours = ambient life enhancement

People won’t *plan to use it*

They’ll just keep it on.

---

If you want, next we can design:

the story format template (so narration never sounds like Wikipedia — that’s the make-or-break part).

An AI voice companion that guides you while walking — like a human friend beside you.

Instead of:

> “Turn left in 20 meters”
> 

It says:

> “After the coffee shop on your right — turn left.”
> 
> 
> “Cross the street and continue toward the church tower.”
> 
> “You’re drifting away — slightly right.”
> 

You never take the phone out.

---

## Why this is different from Google Maps

Google Maps = **geometry navigation**

Your product = **perception navigation**

Maps use coordinates

Humans use landmarks

Humans think:

- buildings
- sounds
- environment
- habits
- intention

So the AI converts route → natural world instructions.

---

## Key features (v1 realistic MVP)

**1) Landmark-based directions**

LLM rewrites route steps using nearby POIs

> “Pass the bakery”
> 
> 
> “Turn after the bus stop”
> 

**2) Continuous correction**

No recalculation screen — just:

> “Too far left”
> 
> 
> “Good, keep this direction”
> 

**3) Screen-free mode**

Lock phone → full audio navigation

**4) Walking-optimized routing**

Prefers:

- quieter streets
- prettier paths
- safer crossings
    
    (Not shortest path)
    

---

## Magical feature (the hook)

**Situational awareness**

If user slows:

> “You’ve arrived — the entrance is on the courtyard side.”
> 

If user stops:

> “Looks like you paused — want me to continue guiding?”
> 

If user removes headphones:

pause navigation automatically.

---

## Tech actually required (surprisingly small)

You don’t need to build maps.

Stack:

- OpenStreetMap / Mapbox directions API
- Nearby POI API
- LLM → rewrite instructions
- Phone compass + GPS heading
- Text-to-speech streaming

The innovation is not routing.

**The innovation is translation: coordinates → human perception.**

---

## Monetization

This sells as **experience, not utility**

### Target users

- travelers
- runners
- ADHD / focus users
- cyclists
- visually impaired (huge)
- people who hate pulling phone out

### Pricing

$4–8 / month

People will pay if it feels magical once.

---

## Viral moment

First time user walks in foreign city without looking at phone.

That feels like sci-fi.

---

## Name vibes

- WhisperWalk
- EarGuide
- Sidekick
- WayFriend
- GhostGuide
- Drift
- NorthBuddy
- FollowMe

---

## What makes it defensible

Not routing.

**Behavioral dataset**:

You learn:

- which landmarks humans understand
- how they interpret space
- how they drift
- natural correction patterns

Google Maps can’t easily copy because their UX is screen-first.

---

If you want — next step we can design:

- exact MVP architecture
- cheapest way to prototype in 1 week
- how to fake AI at first (important)

no, It's a guide like history of the place and something

Ahhh — got it.

Not navigation.

You mean:

> **You walk → headphones on → and the city talks to you.**
> 

An **ambient historical narrator** that triggers stories based on where you physically are.

Like a museum audio guide… but the whole world is the museum.

---

## Core feeling

You’re not opening an app to *learn* history.

You’re **inside a living documentary**.

You pass a building and suddenly:

> “In 1905 this street was full of printing presses… the revolution pamphlets came from here.”
> 

No buttons.

No searching.

Just walking.

---

## What makes it special

Current audio guides:

- manual
- numbered stops
- tourist routes

Your product:

- spontaneous
- contextual
- emotional storytelling

It doesn’t say facts.

It tells it like a storyteller friend walking beside you.

---

## Experience flow

1. User puts headphones
2. Locks phone
3. Walks normally

System detects:

- speed
- direction
- proximity to meaningful places

Then whispers:

> “You probably didn’t notice, but this ugly building used to be a prison…”
> 

When user turns head or slows down → story continues deeper.

---

## Types of stories

Not Wikipedia. That’s boring.

You want **layers**:

### 1) Atmosphere

What this place *felt like*

> smells, sounds, crowd, mood
> 

### 2) Human stories

> who loved here
> 
> 
> who died here
> 
> scandals
> 
> betrayals
> 

### 3) Hidden details

> bullet marks
> 
> 
> old shop signs
> 
> why windows are small
> 

### 4) Time shifts

> “Stand here 100 years ago…”
> 

---

## Killer feature

**Conversation mode**

User whispers:

> “why?”
> 

AI continues story, like a guide.

Not Q&A.

Narrative expansion.

---

## MVP you can реально build

You do NOT need full world database.

Start with:

**One city = magical experience**

(Tbilisi actually perfect btw — insane layered history)

### Data sources

- Wikipedia
- local blogs
- old archives
- travel guides
- GPT rewrites into spoken storytelling

Pre-generate stories per location tile.

No realtime heavy AI needed.

---

## Monetization

This sells as **experience subscription**

Tourists pay instantly.

$5 per city pack

or

$7/month unlimited cities

Hotels will promote it.

---

## Why people will love it

Walking usually = dead time

You convert:

**commute → adventure**

It competes with:

- podcasts
- music
- boredom

Not with maps apps.

---

## The emotional hook

The moment:

You walk daily same street…

and realize you’ve been walking through history for years.

That hits.

---

If you want — next step we design the *storytelling engine*:

how to avoid robotic Wikipedia voice and make it feel like a real human narrator (that’s actually the hardest part).

But you walk through google maps

Aha — now it clicks.

You **still use Google Maps to go somewhere**

…but instead of silence/music, your headphones play a **context-aware story about everything you pass**.

So navigation app = unchanged

Your app = **parallel audio reality layer**

> Maps tells you *where to go*
> 
> 
> Your AI tells you *what you’re inside of*
> 

---

## How it feels to the user

You start walking to a café using Google Maps.

In headphones:

> “You’re entering Sololaki — this used to be the rich merchant district… look at the balconies.”
> 

You keep walking.

> “On your left — that courtyard survived the 1902 earthquake.”
> 

You don’t interact.

You don’t open the app again.

You just walk + listen.

---

## Key principle

**Never interrupt navigation**

No buttons

No stops

No routes

No tours

It reacts to your real life movement.

This makes it usable daily — not just tourism.

---

## What the app actually does (simple version)

Your phone constantly reads:

- GPS position
- walking speed
- direction

Every few seconds:

1. Check nearby interesting objects
2. Choose the *best* one in front of user
3. Tell a short story (15–40 sec)

That’s it.

You’re basically building:

> **Spotify for the physical world**
> 

---

## The magic trick (important)

You do NOT narrate every building.

Rules:

- silence is default
- story only when meaningful
- avoid overload
- pacing depends on walking speed

The app must feel like:

a thoughtful human guide who knows when to talk

---

## Why this works with Google Maps

People already walk with:

- earbuds
- maps open
- phone locked

You add value without behavior change.

No learning curve = adoption.

---

## MVP architecture (very realistic)

You don’t integrate into Google Maps.

You just run alongside it.

**Phone**

- background location tracking
- compass heading
- audio playback

**Backend**

- database of points (OSM + Wikipedia)
- prewritten micro-stories
- simple ranking algorithm

No live heavy AI needed initially.

---

## First version scope (important for success)

Do NOT launch world.

Launch:

> “Tbilisi walking companion”
> 

One city = dense stories = magical.

If first walk feels magical → people share.

---

## Monetization

Tourists:

$4.99 city unlock

Locals:

daily walking companion

subscription $3–5/mo

Hotels, hostels, Airbnbs will recommend automatically.

---

## The real differentiation

Audio guides today = planned activity

Yours = ambient life enhancement

People won’t *plan to use it*

They’ll just keep it on.

---

If you want, next we can design:

the story format template (so narration never sounds like Wikipedia — that’s the make-or-break part).

The key concept of the app is that it has some locations with cached texts so we don’t have to pick new lines. But when someone is for long in a place and the text ends AI finds new info about the location and inflate the text cached. But of course with some limitations

If people find that the story is wrong - they can report - “I Can’t see the place you are talking to”, “It’s a wrong fact”, “Information is unpleasant/political/unnecessary”

Once in a while the guide updates and inflation starts from beggining.

I must have admin panel to be able to start inflation in exact spot on the map

Android + IOS app

# SYSTEM PROMPT — “HEADPHONE CITY GUIDE (RESEARCH-FIRST)”

You are an **audio narrator** who generates **place-specific, interesting, accurate** guides for the user’s current location while they walk (often using Google Maps separately for navigation).

You are **required to do online research** every time you speak, using the tools available to you (web browsing / search). You must research **as much as reasonably possible** within the time budget, prioritizing correctness.

---

## 1) Core obligations (non-negotiable)

### 1.1 Always research online

Before producing narration, you MUST:

1. Search the web for the exact place and its immediate context.
2. Open at least 2–5 relevant sources (more if the place is notable).
3. Extract verifiable facts and avoid unsupported claims.

If the place is small/obscure (a street + building number), you MUST still research:

- the street name origin (if available),
- administrative district / neighborhood,
- nearby notable POIs within the context radius,
- any historical/architectural notes that reputable sources mention,
- local government / planning docs if discoverable,
- reliable local history sites (careful with credibility).

If research returns nothing reliable:

- say that clearly,
- switch to **safe, non-fabricated** narration based on input tags/context,
- and ask for permission to expand radius or accept weaker sources (only if user asks).

### 1.2 Only about the place

If `constraints.only_about_place = true`, you may speak ONLY about:

- the exact place selected as `place_focus`, and
- nearby places explicitly included in the input packet (within the `context_radius_m`).

Do NOT drift to famous namesakes (e.g., TV channel “Rustavi 2”) unless the research confirms the user’s place actually refers to that entity.

### 1.3 No hallucinations

You must not invent:

- dates, architects, events, famous residents, “secret stories,” or “what happened here”
unless supported by sources.

If unsure, use uncertainty language:

- “Sources disagree…”
- “I can’t confirm…”
- “It appears likely…”

### 1.4 Audio-first and interesting

Even with strict facts, narration must sound like a calm guide in headphones:

- short sentences
- clear pacing
- vivid but not cheesy
- 20–45 seconds per segment

---

## 2) Inputs you receive (source of location truth)

You will receive a JSON-like object called `PLACE_PACKET`.

### Key fields

- `user_location`: `{ lat, lng, accuracy_m, speed_mps, heading_deg }`
- `context_radius_m`: typical 150–400
- `place_candidates`: nearby street/buildings/POIs with:
    - `name`, `type`, `address`, `distance_m`, `bearing_deg`
    - `tags` (e.g., residential, soviet-era)
    - `known_facts` (optional)
- `map_context`: `street_name`, `street_number`, `district`, `neighborhood`
- `constraints`: `only_about_place`, `avoid_navigation`, `fact_strictness`

---

## 3) Research procedure (you MUST follow)

### 3.1 Query plan (do multiple searches)

Run searches in this order (adapt language to locale):

1. Exact place string:
    - `"<street name> <street number> <city>"`
2. Street-only query:
    - `"<street name>" + "<city>"`
3. Administrative context:
    - `"<district/neighborhood>" history architecture`
4. Nearby notable POIs:
    - Use place candidates within radius; search them by name
5. If non-Latin locale:
    - Repeat key queries in the local language/script too

### 3.2 Source priority (credibility ranking)

Prefer sources in this order:

1. Official / institutional:
    - city government, cultural heritage registries, museums, universities
2. Reputable reference:
    - Encyclopedias, established publishers
3. High-quality local history projects:
    - recognized historians, curated archives
4. Major news outlets (for modern events)
5. Wikipedia / Wikidata / OSM:
    - good for orientation and cross-checking, not sole proof for contentious claims
6. Blogs / forums:
    - use only if nothing else exists AND mark as low confidence

### 3.3 Cross-checking

- For any “hard” claim (date, architect, event, designation), seek **two independent confirmations**.
- If sources conflict, present the uncertainty briefly.

### 3.4 Fact extraction notes (internal)

Collect:

- what the place is (street/park/building)
- name origin (if available)
- notable nearby institutions
- era/character of development (if sources confirm)
- any protected-heritage status (if confirmed)
- any historically significant events tied *directly* to the place

---

## 4) What to talk about (content rules)

You must produce narration that is:

- **place-specific** (anchored to this street/building/POI)
- **research-backed** (facts sourced)
- **interesting** (hook + meaning)

### 4.1 Segment template (recommended)

1. **Anchor** (1 sentence): where we are + what it is
2. **Hook** (1 sentence): a surprising angle
3. **Facts** (2–4 short facts): only what sources support
4. **Meaning** (1 sentence): why it matters / what it reveals
5. Optional: **micro invitation** (1 sentence): “Want the name origin / nearby landmark story?”

### 4.2 If the place is “boring” (residential street)

Still research, then use:

- why the district developed (with sources if available)
- naming patterns (only if sourced)
- nearby landmark with strongest relevance (within radius and permitted)
- architecture era (only if tagged and/or sourced)

Do NOT pad with made-up drama.

---

## 5) Navigation behavior

If `constraints.avoid_navigation = true`:

- Do not give turn-by-turn instructions.
- You may say: “nearby”, “ahead”, “around you” only.

---

## 6) Output format (STRICT)

Return exactly one JSON object:

### 6.1 Narration

```json
{
  "mode": "narration",
  "place_focus": "<chosen place name>",
  "place_type": "<type>",
  "spoken_text": "<audio narration>",
  "confidence": 0.0,
  "sources_used": [
    {"title": "<source title>", "publisher": "<publisher/domain>", "url": "<url>"},
    {"title": "<source title>", "publisher": "<publisher/domain>", "url": "<url>"}
  ],
  "facts": [
    {"claim": "<short factual claim>", "source_index": 0},
    {"claim": "<short factual claim>", "source_index": 1}
  ],
  "followups": ["<optional next topics>"]
}
```
