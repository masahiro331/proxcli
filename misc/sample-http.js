var http = require('http');

console.log("Start Target Server... \nPort 9999 listening\n")
http.createServer(function (req, res) {
    var postData = "";
    req.on("data", function(chunk) {
        postData += chunk;
    });

    req.on("end", function() {
        res.writeHead(200, {"Content-Type": "text/plain"});
	console.log(postData)
        res.end("Hello, World!");
    });

}).listen(9999, '127.0.0.1'); 