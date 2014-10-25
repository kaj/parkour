var switchtime = new Date()

function showTime() {
    var elapsed = (new Date() - switchtime + 100) / 1000 | 0;
    document.getElementById('currenttime').innerHTML = 
	"Arbetad tid: " + timeString(elapsed) +
	"  Tid kvar till byte: " + timeString(15*60 - elapsed);
}
function timeString(elapsed) {
    var t = "";
    if (elapsed > 60) {
	var minutes = elapsed / 60 | 0
	t = t + minutes + " minuter och ";
	elapsed -= 60 * minutes;
    }
    return t + elapsed + " sekunder."
}

function switchDriver(foo) {
    console.log("Driver is now ... " + foo.target.value);
    switchtime = new Date();
    showTime();
    return false;
}
function pause() {
    console.log("Pause");
    switchtime = new Date();
    driver = null;
    return false;
}

document.getElementsByTagName('button')[0].onclick = switchDriver
document.getElementsByTagName('button')[1].onclick = pause
document.getElementsByTagName('button')[2].onclick = switchDriver

setInterval(showTime, 1000)
