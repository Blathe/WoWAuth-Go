package utils

func ReverseString(str string) string {

	//declare our new string
	newStr := ""
	/* Loop through our string we pass in the function
	Get the reverse of the current i and add that to newstr */
	for i := range str {
		newStr += string(str[(len(str)-i)-1])
	}

	return newStr
}
