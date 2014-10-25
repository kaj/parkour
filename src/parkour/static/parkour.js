var switchtime = new Date()
var driver = null

function showTime() {
    var elapsed = (new Date() - switchtime + 100) / 1000 | 0;
    var msg;
    if (driver) {
	msg = "Arbetad tid: " + timeString(elapsed) +
	    "  Tid kvar till byte: " + timeString(15*60 - elapsed);
    } else {
	msg = "Pause sedan " + timeString(elapsed);
    }
    document.getElementById('currenttime').innerHTML = msg
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
    if (driver != foo.target.value) {
	driver = foo.target.value;
	console.log("Driver is now ", driver);
	switchtime = new Date();
	var req = new XMLHttpRequest();
	req.open('PUT', 'driver');
	req.send(driver);
	showTime();
    }
    return false;
}
function pause() {
    console.log("Pause");
    if (driver) {
	switchtime = new Date();
	driver = null;
	var req = new XMLHttpRequest();
	req.open('PUT', 'pause');
	req.send();
	showTime();
    }
    return false;
}


document.getElementsByTagName('button')[0].onclick = switchDriver
document.getElementsByTagName('button')[1].onclick = pause
document.getElementsByTagName('button')[2].onclick = switchDriver

setInterval(showTime, 1000)
