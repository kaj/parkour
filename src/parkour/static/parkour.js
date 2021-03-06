var switchtime = new Date();
var driver = null;
var targettime = 15 * 60;

function showTime() {
    var elapsed = (new Date() - switchtime + 100) / 1000 | 0;
    var msg;
    if (driver) {
	var remaining = targettime - elapsed;
	if (remaining <= 0.1 * targettime) {
	    document.getElementById('currenttime').className =
		(remaining < 0) ? 'late' : 'soon';
	}
	msg = "since " + timeString(elapsed) +
	    "  Time left to change: " + timeString(remaining);
    } else {
	msg = "since " + timeString(elapsed);
    }
    document.getElementById('currenttime').innerHTML = msg
}

function timeString(elapsed) {
    var t = "";
    if (elapsed > 60) {
	var minutes = elapsed / 60 | 0
	t = t + minutes + " minutes and ";
	elapsed -= 60 * minutes;
    }
    return t + elapsed + " seconds."
}

function switchDriver(foo) {
    console.log("Set driver " + foo.target.value + " from " + driver)
    if (driver != foo.target.value) {
	resetDriver(foo.target.value);
	var req = new XMLHttpRequest();
	req.open('PUT', 'driver');
	req.send(driver);
	showCurrentLog()
    }
    return false;
}

function pause() {
    console.log("Pause from " + driver);
    if (driver) {
	resetDriver(null);
	putPause();
	showCurrentLog()
    }
    return false;
}

function putPause() {
    var req = new XMLHttpRequest();
    req.open('PUT', 'pause', false);
    req.send();
    return true;
}

function resetDriver(d) {
    driver = d;
    switchtime = new Date();
    document.getElementById('currenttime').className = '';
    showTime();
}

function showCurrentLog() {
    $.getJSON("/boutlog", function(data) {
	var items = [], last = "-";
	$.each(data, function(key, val) {
	    item = "<li>"
	    last = val.Entry;
	    if (val.Entry == "pause") {
		item += "Pause";
	    } else {
		item += val.Entry;
		if (val.Duration) {
		    item += " was the driver for " + timeString(val.Duration);
		}
	    }
	    items.unshift(item);
	});
	if (last == "-") {
	    items.unshift("<li>New session");
	} else if (last == "pause") {
	    driver = null;
	} else {
	    items[0] += " is the driver"
	    driver = last;
	}
	items[0] += " <span id='currenttime'/>"
	$("#currentlog ul").replaceWith($( "<ul/>", {
	    html: items.join( "" )
	}))
    })
}


document.getElementsByTagName('button')[0].onclick = switchDriver
document.getElementsByTagName('button')[1].onclick = pause
document.getElementsByTagName('button')[2].onclick = switchDriver
document.getElementById("changebout").onclick = putPause

setInterval(showTime, 1000)
$("form#currentbout").after("<div id='currentlog'><h3>This pair programming session</h3><ul/></div>")
showCurrentLog()
