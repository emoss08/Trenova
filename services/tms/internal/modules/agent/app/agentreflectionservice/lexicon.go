package agentreflectionservice

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	minToldWords = 5
	minToldRunes = 8
)

// lexicon is how one way of writing marks a correction, a standing request, a
// request and a reaction. English and Spanish share one, read by the first
// word of a sentence; Chinese has no spaces, so its openings are prefixes of
// the sentence and its length is counted in characters. People write in the
// language they think in whatever the interface is set to, so the lexicon is
// chosen by the script of the message, not by a locale.
type lexicon struct {
	correctionOpenings []string
	correctionPhrases  []string
	standing           []string
	askingWords        map[string]struct{}
	reactingWords      map[string]struct{}
	askingPrefixes     []string
	reactingPrefixes   []string
	questionEndings    []string
	spaced             bool
}

var latinLexicon = lexicon{
	spaced: true,
	correctionOpenings: []string{
		"no,", "no ", "no.", "nope", "not ", "not that", "wrong", "that's wrong",
		"that is wrong", "that's not", "that is not", "thats not", "actually", "instead",
		"i said", "i meant", "i asked", "don't", "dont", "do not", "stop", "why did you",
		"you should have", "you shouldn't", "you should not", "please don't", "incorrect",
		"that's incorrect", "undo", "redo", "try again",
		"eso no", "así no", "asi no", "incorrecto", "está mal", "esta mal", "mal,",
		"en realidad", "en vez", "mejor ", "dije", "quise decir", "te pedí", "te pedi",
		"no hagas", "deja de", "por qué hiciste", "por que hiciste", "deberías haber",
		"deberias haber", "no debías", "no debias", "deshaz", "inténtalo de nuevo",
		"intentalo de nuevo", "otra vez",
	},
	correctionPhrases: []string{
		"not what i", "i told you", "should have been", "you forgot", "you missed",
		"wrong one", "the wrong", "that isn't right", "that is not right",
		"that's not right", "isn't what", "is not what", "i didn't ask", "i did not ask",
		"no es lo que", "te dije", "debería haber sido", "deberia haber sido",
		"se te olvidó", "se te olvido", "olvidaste", "te faltó", "te falto",
		"el equivocado", "la equivocada", "no está bien", "no esta bien", "no te pedí",
		"no te pedi",
	},
	standing: []string{
		"from now on", "going forward", "in the future", "next time", "every time",
		"each time", "always ", "never ", "by default", "remember that", "remember to",
		"keep in mind", "whenever ", "make sure you", "for future", "from here on",
		"in future",
		"de ahora en adelante", "a partir de ahora", "en adelante", "en el futuro",
		"la próxima vez", "la proxima vez", "cada vez", "siempre ", "nunca ",
		"por defecto", "recuerda que", "recuerda siempre", "ten en cuenta",
		"asegúrate de", "asegurate de", "para el futuro",
	},
	askingWords: stringSet(
		"show", "list", "find", "get", "give", "create", "make", "add", "update", "change",
		"set", "send", "draft", "write", "open", "check", "look", "pull", "run", "book",
		"assign", "cancel", "delete", "remove", "tell", "help", "can", "could", "would",
		"will", "please", "what", "what's", "whats", "when", "where", "who", "who's",
		"whos", "why", "how", "which", "is", "are", "do", "does", "did", "should", "need",
		"want", "let's", "lets", "schedule", "dispatch", "move", "mark", "close", "search",
		"summarize", "explain", "compare", "calculate", "build", "generate", "remind",
		"put", "email", "call", "try", "use",
		"muestra", "muéstrame", "muestrame", "lista", "busca", "encuentra", "dame", "crea",
		"haz", "agrega", "añade", "actualiza", "cambia", "pon", "envía", "envia",
		"redacta", "escribe", "abre", "revisa", "verifica", "mira", "ejecuta", "reserva",
		"asigna", "cancela", "elimina", "borra", "quita", "dime", "ayuda", "ayúdame",
		"ayudame", "puedes", "podrías", "podrias", "puede", "por", "qué", "cuándo",
		"cuando", "dónde", "donde", "quién", "quien", "cómo", "cuál", "cual", "cuánto",
		"cuanto", "cuántos", "cuantos", "hay", "necesito", "quiero", "quisiera",
		"programa", "despacha", "mueve", "marca", "cierra", "resume", "explica",
		"compara", "calcula", "genera", "recuérdame", "recuerdame", "llama", "intenta",
		"usa",
	),
	reactingWords: stringSet(
		"ok", "okay", "thanks", "thank", "great", "perfect", "yes", "yep", "yeah", "no",
		"nope", "sure", "cool", "good", "nice", "got", "sounds", "looks", "that", "that's",
		"thats", "this", "it", "it's", "its", "hi", "hello", "hey", "sorry", "nevermind",
		"never",
		"vale", "gracias", "genial", "perfecto", "sí", "si", "claro", "bien", "bueno",
		"listo", "entendido", "eso", "esto", "hola", "perdón", "perdon", "disculpa",
		"olvídalo", "olvidalo",
	),
	askingPrefixes: []string{
		"i need", "i want", "i'd like", "i would like", "i am looking", "i'm looking",
		"me gustaría", "me gustaria", "estoy buscando",
	},
	questionEndings: []string{"?"},
}

var hanLexicon = lexicon{
	correctionOpenings: []string{
		"不对", "不對", "不是", "错了", "錯了", "不要", "别", "別", "其实", "其實",
		"我说的是", "我說的是", "我是说", "我是說", "应该是", "應該是", "重新", "再试",
		"再試", "撤销", "撤銷",
	},
	correctionPhrases: []string{
		"不是我要的", "我说过", "我說過", "你忘了", "你漏了", "弄错", "弄錯", "搞错",
		"搞錯", "不正确", "不正確",
	},
	standing: []string{
		"以后", "以後", "今后", "今後", "从现在起", "從現在起", "从现在开始", "從現在開始",
		"下次", "每次", "总是", "總是", "永远不要", "永遠不要", "绝不", "絕不", "默认",
		"預設", "记住", "記住", "请记得", "請記得", "务必", "務必", "一律",
	},
	askingPrefixes: []string{
		"请", "請", "帮我", "幫我", "麻烦", "麻煩", "能不能", "可以", "可否", "给我",
		"給我", "查", "显示", "顯示", "列出", "找", "创建", "創建", "建立", "新增", "更新",
		"修改", "发送", "發送", "取消", "删除", "刪除", "安排", "告诉我", "告訴我", "什么",
		"什麼", "为什么", "為什麼", "怎么", "怎麼", "如何", "哪", "谁", "誰", "有没有",
		"有沒有", "是否", "多少", "几", "幾", "我要", "我想", "我需要",
	},
	reactingPrefixes: []string{
		"好", "谢谢", "謝謝", "行", "嗯", "是的", "对", "對", "没问题", "沒問題", "收到",
		"明白", "了解", "你好", "抱歉", "不好意思",
	},
	questionEndings: []string{"?", "？", "吗", "嗎", "呢"},
}

func lexiconOf(text string) *lexicon {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return &hanLexicon
		}
	}

	return &latinLexicon
}

func (l *lexicon) corrects(text string) bool {
	text = strings.TrimLeft(text, "¡¿ ")
	for _, opening := range l.correctionOpenings {
		if strings.HasPrefix(text, opening) {
			return true
		}
	}

	return stringutils.ContainsAny(text, l.correctionPhrases...)
}

func (l *lexicon) states(sentence string) bool {
	if strings.HasPrefix(sentence, "¿") {
		return false
	}
	for _, ending := range l.questionEndings {
		if strings.HasSuffix(sentence, ending) {
			return false
		}
	}
	sentence = strings.TrimLeft(sentence, "¡\"'“”「『(（ ")
	for _, prefix := range l.askingPrefixes {
		if strings.HasPrefix(sentence, prefix) {
			return false
		}
	}
	for _, prefix := range l.reactingPrefixes {
		if strings.HasPrefix(sentence, prefix) {
			return false
		}
	}
	if !l.spaced {
		return utf8.RuneCountInString(sentence) >= minToldRunes
	}

	words := strings.Fields(sentence)
	if len(words) < minToldWords {
		return false
	}
	first := strings.Trim(words[0], ",:;\"'()-")
	if _, asking := l.askingWords[first]; asking {
		return false
	}
	_, reacting := l.reactingWords[first]

	return !reacting
}

func stringSet(words ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(words))
	for _, word := range words {
		set[word] = struct{}{}
	}

	return set
}
