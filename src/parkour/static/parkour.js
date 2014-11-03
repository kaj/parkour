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
	msg = "sedan " + timeString(elapsed) +
	    "  Tid kvar till byte: " + timeString(remaining);
    } else {
	msg = "sedan " + timeString(elapsed);
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
	resetDriver(foo.target.value);
	var req = new XMLHttpRequest();
	req.open('PUT', 'driver');
	req.send(driver);
	showCurrentLog()
    }
    return false;
}

function pause() {
    console.log("Pause");
    if (driver) {
	resetDriver(null);
	var req = new XMLHttpRequest();
	req.open('PUT', 'pause');
	req.send();
	showCurrentLog()
    }
    return false;
}

function resetDriver(d) {
    driver = d;
    switchtime = new Date();
    document.getElementById('currenttime').className = '';
    showTime();
}

function showCurrentLog() {
    $.getJSON("/boutlog", function(data) {
	var items = [];
	$.each(data, function(key, val) {
	    if (val.Duration) {
		items.unshift("<li>" + val.Entry + " " + timeString(val.Duration));
	    } else {
		items.unshift("<li>" + val.Entry);
	    }
	});
	items[0] += " <span id='currenttime'/>"
	$("#currentlog ul").replaceWith($( "<ul/>", {
	    html: items.join( "" )
	}))
    })
}


document.getElementsByTagName('button')[0].onclick = switchDriver
document.getElementsByTagName('button')[1].onclick = pause
document.getElementsByTagName('button')[2].onclick = switchDriver

setInterval(showTime, 1000)
$("form#currentbout").after("<div id='currentlog'><h3>This bout</h3><ul/></div>")
showCurrentLog()
