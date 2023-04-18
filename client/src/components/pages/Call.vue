<script lang="ts" setup>
import { toRef, onMounted, onBeforeUnmount, ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { mediaOptions } from "../../store/DeviceSettingsStore";
import {
  isCallLeft,
  isCallOfferFromInitiator,
  isCallAnswerFromRecipient,
  isCallRequestedReInitialization,
  isUpdateMediaOptions,
} from "../../socketHandling/InterpretEvent";
import useSocketStore from "../../store/SocketStore";
import { useChatMedia } from "../../composables/useChatMedia";
import VidChatUser from "../../components/shared/vidChat/VidChatUser.vue";
import VidChatControls from "../../components/shared/vidChat/VidChatControls.vue";
import {
  CallWebRTCRequestReInitialization,
  CallWebRTCOffer,
  CallWebRTCAnswer,
  CallLeave,
} from "../../socketHandling/OutEvents";
import Peer from "simple-peer";
import useAuthStore from "../../store/AuthStore";

const socketStore = useSocketStore();
const authStore = useAuthStore();

const route = useRoute();
const router = useRouter();
const otherUsersId = toRef(route.params, "id");
const initiator = computed(() => route.query.initiator !== undefined);

const { userStream, displayStream, userMediaStreamID, updateOptions } =
  useChatMedia(negotiateConnection);

function negotiateConnection(isOnMounted?: boolean) {
  gotAnswer.value = false;
  peerUserStreamHasVideo.value = false;
  peerDisplayStreamHasVideo.value = false;
  if (peerInstance.value) {
    peerInstance.value.destroy();
  }
  if (initiator.value) {
    peerUserStream.value = undefined;
    peerDisplayStream.value = undefined;
    makePeer();
  } else if (!isOnMounted) {
    requestReInitialization();
  }
}

function requestReInitialization() {
  socketStore.send({
    event_type: "CALL_WEBRTC_RECIPIENT_REQUEST_REINITIALIZATION",
  } as CallWebRTCRequestReInitialization);
}

const peerUserMediaStreamID = ref("");
const peerInstance = ref<Peer.Instance>();
const peerUserStream = ref<MediaStream>();
const peerUserStreamHasVideo = ref(false);
const peerDisplayStream = ref<MediaStream>();
const peerDisplayStreamHasVideo = ref(false);
const gotAnswer = ref(false);

function initPeer() {
  const peer = new Peer({
    initiator: initiator.value,
    trickle: false,
    streams: [
      ...(userStream.value ? [userStream.value] : []),
      ...(displayStream.value ? [displayStream.value] : []),
    ],
    iceCompleteTimeout: 2000, // 5 seconds is too long
  });
  peer.on("stream", handleStream);
  return peer;
}

// for initializer peer
function makePeer() {
  gotAnswer.value = false;
  const peer = initPeer();
  peer.on("signal", (signal) => {
    if (!gotAnswer.value) {
      socketStore.send({
        event_type: "CALL_WEBRTC_OFFER",
        data: {
          signal: JSON.stringify(signal),
          um_stream_id: userMediaStreamID.value,
          um_vid: mediaOptions.value.userMedia.video,
          dm_vid: mediaOptions.value.displayMedia.video,
        },
      } as CallWebRTCOffer);
    }
  });
  peerInstance.value = peer;
}
// for recipient peer
async function makeAnswerPeer(
  signal: Peer.SignalData,
  userMediaID: string,
  showUserVid: boolean,
  showDisplayVid: boolean
) {
  const peer = initPeer();
  peer.on("signal", (signal) => {
    socketStore.send({
      event_type: "CALL_WEBRTC_ANSWER",
      data: {
        signal: JSON.stringify(signal),
        um_stream_id: userMediaStreamID.value,
        um_vid: mediaOptions.value.userMedia.video,
        dm_vid: mediaOptions.value.displayMedia.video,
      },
    } as CallWebRTCAnswer);
  });
  peerUserMediaStreamID.value = userMediaID;
  peerUserStreamHasVideo.value = showUserVid;
  peerDisplayStreamHasVideo.value = showDisplayVid;
  peer.signal(signal);
  peerInstance.value = peer;
}

async function signalAnswer(
  signal: Peer.SignalData,
  userMediaID: string,
  showUserVid: boolean,
  showDisplayVid: boolean
) {
  gotAnswer.value = true;
  peerUserMediaStreamID.value = userMediaID;
  peerUserStreamHasVideo.value = showUserVid;
  peerDisplayStreamHasVideo.value = showDisplayVid;
  peerInstance.value?.signal(signal);
}

function watchForCallEvents(e: MessageEvent) {
  const msg = JSON.parse(e.data);
  if (!msg) return;
  if (isCallLeft(msg)) {
    peerInstance.value?.destroy();
    router.push("/");
  }
  if (isCallOfferFromInitiator(msg)) {
    makeAnswerPeer(
      JSON.parse(msg.data.signal) as Peer.SignalData,
      msg.data.um_stream_id,
      msg.data.um_vid,
      msg.data.dm_vid
    );
  }
  if (isCallAnswerFromRecipient(msg)) {
    signalAnswer(
      JSON.parse(msg.data.signal) as Peer.SignalData,
      msg.data.um_stream_id,
      msg.data.um_vid,
      msg.data.dm_vid
    );
  }
  if (isCallRequestedReInitialization(msg)) {
    negotiateConnection();
  }
  if (isUpdateMediaOptions(msg)) {
    peerUserStreamHasVideo.value = msg.data.um_vid;
    peerDisplayStreamHasVideo.value = msg.data.dm_vid;
    peerUserMediaStreamID.value = msg.data.um_stream_id;
  }
}

function handleStream(stream: MediaStream) {
  if (stream.id === peerUserMediaStreamID.value) {
    peerUserStream.value = stream;
  } else {
    peerDisplayStream.value = stream;
  }
}

onMounted(() => {
  socketStore.socket?.addEventListener("message", watchForCallEvents);
});
onBeforeUnmount(() => {
  socketStore.socket?.removeEventListener("message", watchForCallEvents);
  socketStore.send({
    event_type: "CALL_LEAVE",
  } as CallLeave);
});
</script>

<template>
  <div class="container">
    <div class="vid-chat-users">
      <!-- Current user -->
      <VidChatUser
        :userMediaStreamID="userMediaStreamID"
        :uid="authStore.uid"
        :userMedia="userStream"
        :displayMedia="displayStream"
        :isOwner="true"
        :mediaOptions="mediaOptions"
        :hasDisplayMediaVideo="mediaOptions.displayMedia.video"
        :hasUserMediaVideo="mediaOptions.userMedia.video"
      />
      <!-- Other user -->
      <VidChatUser
        :userMedia="peerUserStream"
        :displayMedia="peerDisplayStream"
        :userMediaStreamID="peerUserMediaStreamID"
        :isOwner="false"
        :uid="String(otherUsersId)"
        :hasDisplayMediaVideo="peerDisplayStreamHasVideo"
        :hasUserMediaVideo="peerUserStreamHasVideo"
      />
    </div>
    <VidChatControls
      :updateOptions="updateOptions"
      :exitButtonClicked="() => router.push('/')"
    />
  </div>
</template>

<style lang="scss" scoped>
.container {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  width: 100%;
  height: 100%;
  .vid-chat-users {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    padding: 0.6rem;
    flex-shrink: 1;
  }
}
</style>
