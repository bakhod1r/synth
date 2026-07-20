# Synth — Core Design

**Sana:** 2026-07-20
**Status:** Tasdiqlangan (yadro interfeys)

## Maqsad

Synth — sof **data provayder** Go kutubxonasi. Chaqiruvchi (seeder tool, test, API mock handler) model beradi, Synth realistik, o'zaro bog'liq yozuvlar qaytaradi.

**Chegaralar (qat'iy):**
- Synth DB'ga **ulanmaydi**. DSN qabul qilmaydi, tarmoqqa chiqmaydi.
- Synth **INSERT qilmaydi**. Yuklashni boshqa tool bajaradi.
- Synth DDL/migration parse **qilmaydi**. Model chaqiruvchidan keladi.
- Synth sof: bir xil seed → bir xil natija.

## 1. Yadro interfeys

```go
users  := synth.Make[User](10_000)
orders := synth.Make[Order](500_000, synth.Ref(users, "UserID"))
```

Model — oddiy Go struct:

```go
type User struct {
    ID        uuid.UUID `synth:"pk"`
    Name      string    `synth:"name"`
    Email     string    `synth:"email,from=Name"`
    Phone     string    `synth:"phone"`
    City      string    `synth:"city"`
    Postcode  string    `synth:"postcode,match=City"`
    CreatedAt time.Time `synth:"time,range=2y"`
}
```

**Tag ixtiyoriy** — batafsili 1a-bo'limda.

Uch ajratuvchi xususiyat:
- `from=Name` — hosilaviy maydon: email ismdan yasaladi
- `match=City` — koherentlik: postcode shaharga mos
- `Ref(parent, "FK")` — referensial yaxlitlik: FK haqiqiy parent'ga tegadi

## 1a. Avtomatik xulosa (tag yozmasdan)

**Asosiy holat — tagsiz struct.** Foydalanuvchi hech narsa yozmasdan mazmunli data olishi kerak:

```go
type User struct {
    ID        uuid.UUID
    FullName  string
    Email     string
    Phone     string
    City      string
    Postcode  string
    Age       int
    IsActive  bool
    CreatedAt time.Time
}

users := synth.Make[User](1000)   // tag yo'q, lekin data koherent
```

Xulosa uch bosqichda, shu tartibda — birinchi mos kelgani yutadi:

**1-bosqich: maydon nomi.** Nom normallashtiriladi (`FullName`, `full_name`, `fullname` → `fullname`) va provayder nomlari lug'atiga solishtiriladi. Lug'at sinonimlarni biladi: `fullname`/`name`/`username` → ism, `mail`/`email` → email, `tel`/`phone`/`mobile` → telefon, `zip`/`postcode`/`postal` → postcode. Bu matching sinonim jadvali orqali, fuzzy emas — taxmin qilinadigan va sozlanadigan.

**2-bosqich: Go turi.** Nom hech narsa demasa, tur qaror qiladi: `time.Time` → yaqin o'tmishdagi vaqt, `bool` → 50/50, `uuid.UUID` → v4, `int` → 0–1000, `float64` → 0–1000.0, `string` → qisqa lorem, `[]T`/`*T` → rekursiv (pointer'lar ~10% `nil`).

**3-bosqich: taslim bo'lish.** Ikkalasi ham ishlamasa (masalan noma'lum `struct` turi) — maydon nol qiymatda qoldiriladi va **ogohlantirish** yig'iladi. Jim qolmaydi: `synth.Warnings(users)` qaysi maydonlar to'ldirilmagani va nima uchunligini qaytaradi.

**Bog'liqliklar ham avtomatik topiladi.** Xulosa bosqichida ma'lum juftliklar avtomatik ulanadi, tag yozmasdan:
- Ism maydoni + email maydoni bo'lsa → email ismdan hosil qilinadi
- Shahar + postcode/region bo'lsa → bir-biriga mos qilinadi
- Bir nechta `time.Time` bo'lsa → nom semantikasidan tartib chiqariladi (`created` < `updated` < `deleted`)

Bu qoidalar `synth.Infer` konfiguratsiyasida ochiq: foydalanuvchi o'z sinonimlarini qo'sha oladi (`synth.Alias("ismi", "name")`) — masalan o'zbekcha maydon nomlari uchun.

**Tag har doim ustun.** Tag bo'lsa — xulosa umuman ishlamaydi, tag aynan bajariladi. Ya'ni tag = xulosani bekor qilish mexanizmi, majburiyat emas.

## 2. Arxitektura

Uch qatlam, aniq chegaralar bilan:

```
  Frontend       →        IR        →      Engine
(struct+tag)          (Schema)          (generatorlar)
```

**`schema` paketi** — IR. Struct'dan mustaqil ma'lumot tuzilmasi: maydonlar ro'yxati, har biri `Kind`, parametrlar, bog'liqliklar (`from`, `match`) bilan. Engine faqat shuni biladi.

**`reflectfe` paketi** — yagona frontend: Go struct + tag → `schema.Schema`. Reflection natijasi tur bo'yicha keshlanadi (har chaqiruvda emas, bir marta).

**`infer` paketi** — 1a-bo'limdagi xulosa qoidalari. `reflectfe` tagsiz maydonga duch kelganda shuni chaqiradi. Alohida paket, chunki qoidalar mustaqil test qilinadi va foydalanuvchi ularni kengaytira oladi.

**`gen` paketi** — engine. `schema.Schema` + `rand.Source` → yozuvlar. Struct haqida hech narsa bilmaydi.

**`providers` paketi** — atomik qiymat generatorlari (`name`, `email`, `iban`, `card`, `city`...). Har biri kichik interfeys: `Generate(ctx, rng) any`. Yangi tur qo'shish = bitta fayl qo'shish, boshqa hech narsaga tegmaslik.

**`locale` paketi** — `uz_UZ`, `en_US`, `ru_RU` uchun ma'lumot to'plamlari (ismlar, shaharlar+postcode juftliklari, telefon prefikslari, valyuta). Data — embed qilingan fayllar, kod emas.

Bu chegaralar keyinchalik YAML frontend yoki CLI qo'shishni arzon qiladi: ular ham shunchaki `schema.Schema` ishlab chiqaradi.

## 3. Bog'liqlik hal qilish (dependency resolution)

`from=` va `match=` maydonlar orasida graf hosil qiladi. Engine har bir tur uchun **bir marta** topologik tartib hisoblaydi va keshlaydi, keyin har yozuvni shu tartibda to'ldiradi — `Name` `Email`dan oldin.

Sikl aniqlansa (`A from=B`, `B from=A`) — `Make` panic emas, **aniq xatolik** qaytaradi, qaysi maydonlar sikl hosil qilgani ko'rsatilgan holda.

## 4. Ref va kardinallik

```go
synth.Ref(users, "UserID")                      // tasodifiy parent
synth.Ref(users, "UserID", synth.OneToMany(3,7)) // har parent'ga 3–7 ta child
```

`Ref` parent slice'dan PK qiymatlarini bir marta chiqarib oladi, keyin child'larga taqsimlaydi. Parent bo'sh bo'lsa — xatolik, jim qolish emas.

## 5. Temporal sababiyat

Bir yozuv ichida vaqt maydonlari tartibni hurmat qiladi: `CreatedAt < PaidAt < ShippedAt`. Tag orqali:

```go
PaidAt time.Time `synth:"time,after=CreatedAt,gap=1h..48h"`
```

`after=` — bu `from=` bilan bir xil graf mexanizmi, alohida tizim emas.

## 5a. Faylga yozish (encoders)

`Make` `[]T` qaytaradi — kod ichida ishlatish uchun. Lekin ko'p holatda natija faylga kerak (seeder o'qishi, qo'lda tekshirish uchun). Shuning uchun `synth.Write`:

```go
users := synth.Make[User](10_000)
synth.WriteCSV("users.csv", users)          // CSV
synth.WriteJSONL("users.jsonl", users)      // har qatorda bir JSON
synth.WriteSQL("users.sql", users, "users") // INSERT statement'lar
```

Yoki to'g'ridan-to'g'ri generatsiya + yozish, oraliq slice'siz (millionlab qator uchun, doimiy xotira):

```go
synth.Stream[User](1_000_000).ToCSV("users.csv")
```

**`encode` paketi** — encoder'lar. Har biri `io.Writer` va yozuvlar oqimini oladi:
- **CSV** — sarlavha struct maydon nomlaridan; ustuncha tartibi barqaror.
- **JSONL** — `json.Marshal`, har yozuv bir qator; stream uchun ideal.
- **SQL** — parametrsiz `INSERT INTO t (...) VALUES (...)`; qiymatlar to'g'ri escape qilinadi. Bu **fayl**, DB ulanishi emas — seeder shu faylni o'zi ishga tushiradi.

Encoder'lar `gen` engine'idan mustaqil: ular faqat maydon nomlari + qiymatlarni ko'radi, shuning uchun yangi format (Parquet, XML) qo'shish = bitta fayl. `Stream` esa `gen`'dan yozuvlarni bittalab oladi va darrov encoder'ga uzatadi — hech qachon hammasi xotirada turmaydi.

## 6. Xatoliklar

Ikki toifa, aniq ajratilgan:

- **Konfiguratsiya xatoliklari** (noma'lum tag, sikl, mos kelmaydigan tur) — `Make` `error` qaytaradi. Bular dasturchi xatosi va tez, aniq xabar bilan chiqishi kerak.
- **Generatsiya paytidagi xatolik yo'q.** Provayderlar har doim qiymat qaytaradi. Agar provayder qiymat bera olmasa, bu konfiguratsiya xatosi va `Make` boshida aniqlanadi (validatsiya generatsiyadan oldin, alohida bosqichda).

`Make` ikki shaklda: `Make[T](n, ...)` — xatoda panic, tez prototip uchun; `TryMake[T](n, ...) ([]T, error)` — ishlab chiqarish uchun.

## 7. Determinizm

```go
synth.WithSeed(42)
```

Bir xil seed + bir xil schema → bayt-bayt bir xil natija. Har bir yozuv o'z `rng`ini seed'dan hosil qiladi, shuning uchun parallel generatsiya ham determinist qoladi.

## 8. Testlash

- **Provayderlar** — property-based: har bir IBAN mod-97 dan o'tadi, har bar karta Luhn'dan, har telefon E.164'ga mos. Misol emas, xususiyat tekshiriladi.
- **Graf hal qilish** — jadval testlari: to'g'ri tartib, sikl aniqlash, chuqur zanjirlar.
- **Koherentlik** — `match=City` uchun: 10k yozuv generatsiya qilinadi, har birining postcode'i shahriga tegishli ekani tekshiriladi.
- **Xulosa** — jadval testlari: har bir sinonim to'g'ri provayderga tushishi, tur-fallback'lari, tag xulosani bekor qilishi, va tagsiz struct uchun ogohlantirishlar to'g'ri yig'ilishi.
- **Encoder'lar** — round-trip: CSV/JSONL yozib, qayta o'qib, yozuvlar teng ekani; SQL escape'i (apostrof, `NULL`) to'g'ri; `Stream` xotira o'smasligi (1M qator doimiy xotirada).
- **Determinizm** — bir xil seed bilan ikki marta chaqirib, natijalar solishtiriladi.
- **Reflection keshi** — bir tur uchun `Make` ikki marta chaqirilsa, reflection bir marta ishlashi.

## Ko'lamdan tashqarida (hozircha)

DB ulanish, INSERT, sink'lar (Kafka/Postgres), DDL parsing, web UI, YAML schema, OpenAPI. Bular keyingi spec'larda — yadro IR ularni arzon qiladi.
