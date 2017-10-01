<template>
  <div>
    <div class="left">
      <h3>Cameras</h3>
      <ul>
        <li v-for="cam in cams" :key="cam.id">
          <router-link  v-bind:to="`/play/${cam.id}/${playTimeFormatted}`">{{cam.name}}</router-link>
        </li>
      </ul>

      <h3>Days</h3>
      <ul>
        <li v-for="day in days">
          <router-link  v-bind:to="`/play/${$route.params.cam}/${day.format('YYYYMMDDHHmmss')}`">{{day | date}}</router-link>
        </li>
      </ul>
    </div>
    <div class="player">
      <video v-show="!error" ref="video" width="100%" height="100%" autoplay controls v-on:timeupdate="timeUpdate"></video>
      <div v-show="error" class="player-error">
        Error {{error}} :(
      </div>
    </div>
    <div class="controls">
      <div class="controls-buttons">
        {{playTime | time}}
      </div>
      <div class="controls-playbar">
        <svg ref="playbar" class="playbar" v-on:mousemove="playbarMouseMove">
          <rect width="100%" height="100%" style="fill:#000;" />
          <rect v-for="p in periods"
            v-bind:x="(p.time - startTime) / (endTime - startTime) * 100 + '%'"
            v-bind:width="p.duration / (endTime - startTime) * 100 + '%'"
            height="100%" style="fill:#36C;" />
        </svg>
        <svg ref="playhead" class="playhead" width="10" height="20"
          v-on:mousemove="playbarMouseMove"
          v-bind:style="{left: playTime ? ((playTime - startTime) / (endTime - startTime) * 100 + '%') : 0}"
          >
          <path d="M 5 0 L 10 5 V 20 H 0 V 5 Z" style="fill:#fff"/>
        </svg>
      </div>
    </div>
  </div>
</template>

<script>
import router from './../router'
import moment from 'moment'
import shaka from 'shaka-player'

export default {
  name: 'player',
  data () {
    return {
      msg: 'Welcome to Your Vue.js App',
      periods: [],
      startTime: null,
      endTime: null,
      playTime: null,
      error: null,
      cams: []
    }
  },
  computed: {
    playTimeFormatted () {
      return moment(this.playTime).format('YYYYMMDDHHmmss')
    },
    days () {
      var res = []
      for (var i = 0; i < 30; i++) {
        var v = moment().startOf('day').subtract(i, 'd')
        res.push(v)
      }
      return res
    }
  },
  methods: {
    timeUpdate (e) {
      if (!this.updateUrl) {
        return
      }
      var t = this.$refs.video.currentTime * 1000
      for (var p of this.periods) {
        if (t >= p.startTime && t <= p.startTime + p.duration) {
          this.playTime = p.time + t - p.startTime
        }
      }
      router.replace(`/play/${this.$route.params.cam}/${moment(this.playTime).format('YYYYMMDDHHmmss')}`)
    },
    playbarMouseMove (e) {
    },
    loadVideo (cam, time) {
      if (time) {
        this.playTime = +moment(time, 'YYYYMMDDHHmmss')
        this.startTime = +moment(time, 'YYYYMMDDHHmmss').startOf('day')
      } else {
        this.playTime = +moment()
        this.startTime = +moment().startOf('day')
      }
      this.endTime = moment(this.startTime).add(1, 'd')
      this.periods = []

      var manifestUri = `/mpd?cam=${cam}&from=${this.startTime}&to=${this.endTime}`
      this.periods = []
      this.updateUrl = false
      this.error = null
      this.player.load(manifestUri).then(() => {
        console.log('The video has now been loaded!')

        // Seek video to the specified time.
        for (var p of this.periods) {
          var t = 0
          if (this.playTime >= p.time && this.playTime <= p.time + p.duration) {
            t = (this.playTime + p.startTime - p.time) * 0.001
          }
          setTimeout(() => {
            this.$refs.video.currentTime = t
            this.updateUrl = true
          }, 300)
        }
      }).catch((error) => {
        console.error('LOAD VIDEO error code', error.code, 'object', error)
        this.error = error.code
      })
    }
  },
  beforeRouteUpdate (to, from, next) {
    var timeReload = false
    if (to.params.time) {
      var time = +moment(to.params.time, 'YYYYMMDDHHmmss')
      if (Math.abs(time - this.playTime) > 2000) {
        timeReload = true
      }
    }
    if (to.params.cam !== this.$route.params.cam || timeReload) {
      this.loadVideo(to.params.cam, to.params.time)
    }
    next()
  },
  mounted () {
    fetch('/cameras', {
      credentials: 'include'
    }).then((response) => {
      return response.json()
    }).then((cams) => {
      this.cams = cams
    })
    .catch((error) => {
      console.log(error)
    })

    function onErrorEvent (event) {
      // Extract the shaka.util.Error object from the event.
      onError(event.detail)
    }

    function onError (error) {
      console.error('Error code', error.code, 'object', error)
    }

    // Create a Player instance.
    var video = this.$refs.video
    var player = new shaka.Player(video)

    window.player = player
    this.player = player

    // Listen for error events.
    player.addEventListener('timelineregionadded', (e) => {
      var p = {
        time: +moment(e.detail.value),
        startTime: e.detail.startTime * 1000,
        duration: (e.detail.endTime - e.detail.startTime) * 1000
      }
      this.periods.push(p)
    })

    player.addEventListener('error', onErrorEvent)

    this.loadVideo(this.$route.params.cam, this.$route.params.time)
  }
}
</script>

<style scoped>
.left {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 160px;
}
.player {
  position: absolute;
  left: 160px;
  top: 0;
  bottom: 70px;
  right: 0;
  background: black;
}
.player-error {
  margin-top: 30%;
  text-align: center;
  font-size: 30px;
}
.controls {
  position: absolute;
  left: 160px;
  height: 70px;
  bottom: 0;
  right: 0;

  background: #333;

  display: flex;
}

.controls-buttons {
}
.controls-playbar {
  position: relative;
  flex-grow: 1;
}
.controls-playbar .playbar {
  width: 100%;
  height: 20px;
  border: 1px solid #555;
  border-radius: 7px;
}
.controls-playbar .playhead {
  position: absolute;
  top: 10px;
  transform: translateX(-50%);
}

</style>
