// Package seed generates the demo data: 5 outlets and 1,500 reviews over 26
// weeks with a planted wait-time spike (US-02-006, HLD flow D). Everything is
// deterministic: the same week gives the same texts in the same order, so
// review ids, batches and therefore the recorded tagging answers match on
// every seed. Only the dates move with the week.
package seed

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// Outlets are the demo outlets, in the order they are created.
var Outlets = []string{"Indiranagar", "Koramangala", "HSR Layout", "Whitefield", "Jayanagar"}

// SpikeOutlet is the outlet whose wait-time complaints spike in the latest
// complete week (REQ-043).
const SpikeOutlet = 1 // Koramangala

// Numbers from the backlog and the HLD (section 8).
const (
	Total      = 1500
	Weeks      = 26
	SpikeExtra = 15
)

// MinTotal is the smallest set GenerateN makes: one review per outlet and
// week, plus the spike.
var MinTotal = len(Outlets)*Weeks + SpikeExtra

// Review is one generated review; Outlet indexes Outlets.
type Review struct {
	Outlet       int
	Source       string
	Date         time.Time
	Rating       int
	Text         string
	ReviewerName string
}

// Generate returns the Total reviews for the 26 weeks ending with the week
// that starts on latestMonday, ordered by date then outlet.
func Generate(latestMonday time.Time) []Review { return GenerateN(latestMonday, Total) }

// GenerateN is Generate with a smaller or larger set, for trial runs on a
// rate-limited model (make seed SEED_ARGS=-reviews=500). The spike is the
// same 15 reviews whatever the total; total must be at least MinTotal.
func GenerateN(latestMonday time.Time, total int) []Review {
	rng := rand.New(rand.NewPCG(20261006, 1500))
	cells := len(Outlets) * Weeks
	base := (total - SpikeExtra) / cells
	extra := (total - SpikeExtra) % cells
	firstMonday := latestMonday.AddDate(0, 0, -7*(Weeks-1))

	var out []Review
	n := 0
	for w := range Weeks {
		monday := firstMonday.AddDate(0, 0, 7*w)
		for o := range Outlets {
			count := base
			if w*len(Outlets)+o < extra {
				count++
			}
			for range count {
				out = append(out, pick(rng, n, o, monday))
				n++
			}
			if w == Weeks-1 && o == SpikeOutlet {
				for i := range SpikeExtra {
					r := spike(rng, n, i, o, monday)
					out = append(out, r)
					n++
				}
			}
		}
	}
	return out
}

var sources = []string{"Google", "Google", "Zomato", "Swiggy"}

var firstNames = []string{
	"Asha", "Ravi", "Priya", "Karan", "Meera", "Arjun", "Sunita", "Vikram", "Neha", "Rahul",
	"Ananya", "Rohan", "Kavya", "Aditya", "Isha", "Siddharth", "Pooja", "Manish", "Divya", "Nikhil",
	"Shreya", "Varun", "Aditi", "Harsh", "Tanvi", "Kabir", "Riya", "Aman", "Sneha", "Yash",
	"Lakshmi", "Gopal", "Fatima", "Imran", "Zoya", "Farhan", "Deepa", "Suresh", "Anjali", "Mohit",
	"Bhavna", "Tarun", "Nisha", "Pranav", "Swati", "Kunal", "Ritu", "Sameer", "Jaya", "Vivek",
	"Geeta", "Alok", "Payal", "Naveen", "Smita", "Ashok", "Rekha", "Dev", "Maya", "Ishaan",
}

// name is unique per review number, so no two reviews share the natural
// key (outlet, source, date, reviewer, text).
func name(n int) string {
	return fmt.Sprintf("%s %c.", firstNames[n%len(firstNames)], 'A'+rune(n/len(firstNames))%26)
}

func day(rng *rand.Rand, monday time.Time) time.Time { return monday.AddDate(0, 0, rng.IntN(7)) }

// pick draws one ordinary review: mostly positive, some neutral, some
// negative, a few urgent; English, Hindi in Devanagari and Hinglish.
func pick(rng *rand.Rand, n, outlet int, monday time.Time) Review {
	r := Review{Outlet: outlet, Source: sources[rng.IntN(len(sources))], Date: day(rng, monday), ReviewerName: name(n)}
	switch p := rng.IntN(100); {
	case p < 52:
		r.Text, r.Rating = positive[rng.IntN(len(positive))], 4+rng.IntN(2)
	case p < 67:
		r.Text, r.Rating = neutral[rng.IntN(len(neutral))], 3
	case p < 97:
		r.Text, r.Rating = negative[rng.IntN(len(negative))], 1+rng.IntN(2)
	default:
		r.Text, r.Rating = urgent[rng.IntN(len(urgent))], 1
	}
	return r
}

// spike is one of the planted wait-time complaints in the latest week.
func spike(rng *rand.Rand, n, i, outlet int, monday time.Time) Review {
	return Review{Outlet: outlet, Source: sources[rng.IntN(len(sources))], Date: day(rng, monday), Rating: 1 + rng.IntN(2),
		Text: waits[i%len(waits)], ReviewerName: name(n)}
}

var positive = []string{
	"Lovely dosa and the filter coffee was perfect. Staff were warm and quick.",
	"Great food at a fair price. The paneer tikka was smoky and fresh.",
	"Clean place, friendly staff, and our order came out in ten minutes.",
	"Best biryani in the area. Generous portion and good value for money.",
	"Came with family on a Sunday. Kids loved the food and the staff were patient.",
	"खाना बहुत स्वादिष्ट था और स्टाफ बहुत अच्छा था। फिर आएँगे।",
	"साफ़-सुथरी जगह, गरम खाना और सही दाम। बहुत बढ़िया अनुभव रहा।",
	"Khana ekdum mast tha, service bhi fast thi. Will come again for sure.",
	"Paneer butter masala was amazing yaar, and the price was very reasonable.",
	"Quick service even at lunch rush. The thali was fresh and filling.",
	"Staff remembered our order from last time. Lovely touch, great chai.",
	"Spotless washrooms and tables, and the masala dosa was crisp.",
}

var neutral = []string{
	"Food was okay, nothing special. Service was average.",
	"Decent place for a quick bite. Prices are as expected.",
	"The curry was good but the naan was a bit dry. Staff were fine.",
	"खाना ठीक-ठाक था। जगह साफ़ थी पर कुछ खास नहीं।",
	"Theek thak experience. Food average tha, service normal.",
	"Good coffee, average snacks. Seating is a bit cramped.",
}

var negative = []string{
	"The food was cold and bland. Very disappointed with the dal.",
	"Way too expensive for such small portions. Not worth the money.",
	"Tables were sticky and the floor was dirty. Needs better cleaning.",
	"The waiter was rude and ignored us when we asked for water.",
	"We waited 40 minutes for our food and nobody told us why.",
	"Overpriced and the biryani was dry. The staff did not seem to care.",
	"खाना ठंडा था और बहुत महंगा भी। पैसे बर्बाद हुए।",
	"स्टाफ का व्यवहार अच्छा नहीं था और टेबल गंदी थी।",
	"Bahut wait karna pada, 45 minute lag gaye order aane mein.",
	"Khana bekaar tha aur price bhi zyada. Paisa vasool nahi.",
	"Washroom was filthy and there were flies near the counter.",
	"Ordered paneer, got something else, and the manager was dismissive.",
}

var urgent = []string{
	"Found a cockroach in my biryani. My son threw up that night. This is a hygiene hazard.",
	"After eating the chicken, two of us had food poisoning the next day. The meat smelled off.",
	"A staff member kept making remarks about my clothes after I asked him to stop.",
	"The manager shouted at my mother and pushed her chair. Completely unacceptable.",
	"If I don't get a refund for the spoiled food by Friday, I will file a complaint in consumer court.",
	"खाने में कीड़ा मिला और रात को मेरे बेटे का पेट खराब हो गया। रसोई की सफ़ाई की जाँच कीजिए।",
	"Staff ne meri dost pe gande comments kiye. Main police complaint karungi.",
	"I am reporting this outlet to FSSAI. The paneer was rotten and they still served it.",
}

var waits = []string{
	"Waited over an hour for our food even though the place was half empty.",
	"45 minutes for two dosas. Nobody came to tell us what was happening.",
	"We waited 50 minutes for a table even with a booking, and the food took another 30.",
	"Order took forever. We asked three times and still waited 40 minutes.",
	"Bahut lamba wait tha yaar, ek ghanta laga khana aane mein.",
	"खाना आने में एक घंटा लग गया। बहुत इंतज़ार करवाया।",
	"Slowest service ever. Our starters came after 35 minutes and mains after an hour.",
	"Waited 40 minutes for the bill alone. The staff kept ignoring us.",
}
