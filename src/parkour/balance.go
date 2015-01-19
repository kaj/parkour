package parkour

type Balance struct {
    MyName string
    MySeconds int
    OthersNames string
    OthersSeconds int
}

func (b *Balance) MyPct() int {
    total := b.MySeconds + b.OthersSeconds
    if total > 0 {
        return 100 * b.MySeconds / total
    } else {
        return 50;
    }
}
func (b *Balance) OthersPct() int {
    total := b.MySeconds + b.OthersSeconds
    if total > 0 {
        return 100 * b.OthersSeconds / total
    } else {
        return 50;
    }
}

func (b *Balance) MyDescr() string {
    return GetUser(b.MyName).Firstname + " " + formatSeconds(b.MySeconds)
}

func (b *Balance) OthersDescr() string {
    return GetUser(b.OthersNames).Firstname + " " + formatSeconds(b.OthersSeconds)
}
