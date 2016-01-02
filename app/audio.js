  var threshold = 15
  var smoothingTimeConstant = 0.1
  var fftSize = 1024

  fetchAudio('../audio/02_How_Did_I_Get_Here.wav').then((data) => {
        var context = new AudioContext()
        if(data instanceof ArrayBuffer) {
          context.decodeAudioData(data, (buffer) => {
            connectAndStart(buffer, context)
          })
        } else {
          connectAndStart(data, context)
        }
  })

  function fetchAudio(url) {
    return new Promise(function(resolve) {
      var request = new XMLHttpRequest()
      request.open('GET', url, true)
      request.responseType = 'arraybuffer'
      request.onload = function() {
        resolve(request.response)
      }
      request.send()
    })
  }

  setTimeout(function() {
              console.log(times)
          }, 6000)

  var analyser
  function connectAndStart(buffer, context) {
        var source = context.createBufferSource()
        analyser = context.createAnalyser()
        analyser.fftSize = 1024
        analyser.smoothingTimeConstant = 0.8
        source.buffer = buffer
        source.connect(analyser)
        source.connect(context.destination)
        lastTime = new Date().getTime()
        source.start(0, 0)
        compareFrames()
  }

  var times = ''
  var lastTime
  var lastFrameVal = 0
  function compareFrames() {
    setFrameInterval()
    var curFrameVal = getCurrentFrameVal()
    var change = Math.abs(curFrameVal - lastFrameVal)
    lastFrameVal = curFrameVal
    if(change > threshold) {
      var currentTime = new Date().getTime()
      var diff = currentTime - lastTime
      times += diff + ', '
      lastTime = currentTime
    }
  }

    function getCurrentFrameVal() {
      var frequencyData = new Uint8Array(analyser.frequencyBinCount); //empty array
      analyser.getByteFrequencyData(frequencyData); //populated array
      return getAvgVolume(frequencyData);
    }

    function getAvgVolume(frequencyData){
        var values = 0;
        var m = {}
        frequencyData.forEach(function(val) {
          values += val;
        })
        return values/frequencyData.length;
    }

    var frameInterval
    function   setFrameInterval() {
     if (!frameInterval) {
       frameInterval = setInterval(()=> {
         compareFrames();
       }, 1000/24)
     }
   }