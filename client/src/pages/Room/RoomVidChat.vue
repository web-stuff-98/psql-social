<script lang="ts" setup>
import { nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import { useChatMedia } from "../../composables/useChatMedia";
import { mediaOptions } from "../../store/DeviceSettingsStore";
import VidChatControls from "../../components/shared/vidChat/VidChatControls.vue";
import VidChatUser from "../../components/shared/vidChat/VidChatUser.vue";
import useRoomChannelStore from "../../store/RoomChannelStore";
import useAuthStore from "../../store/AuthStore";
import useSocketStore from "../../store/SocketStore";
import Peer from "simple-peer";
import {
  ChannelWebRTCLeave,
  ChannelWebRTCSendingSignal,
  ChannelWebRTCReturningSignal,
  ChannelWebRTCJoin,
} from "../../socketHandling/OutEvents";
import {
  isChannelWebRTCAllUsers,
  isChannelWebRTCReceivingReturnedSignal,
  isChannelWebRTCUserJoined,
  isChannelWebRTCUserLeft,
  isUpdateMediaOptions,
} from "../../socketHandling/InterpretEvent";

defineProps<{
  exitButtonClicked: Function;
}>();

const socketStore = useSocketStore();
const authStore = useAuthStore();
const roomChannelStore = useRoomChannelStore();

const { userStream, displayStream, userMediaStreamID, updateOptions } =
  useChatMedia(() => {
    peersData.value.forEach((p) => p.peer.destroy());
    peersData.value = [];
    socketStore.send({
      event_type: "CHANNEL_WEBRTC_JOIN",
      data: {
        channel_id: roomChannelStore.current,
        um_stream_id: userMediaStreamID.value,
        um_vid: mediaOptions.value.userMedia.video,
        dm_vid: mediaOptions.value.displayMedia.video,
      },
    } as ChannelWebRTCJoin);
  }, roomChannelStore.current);

type PeerData = {
  peer: Peer.Instance;
  userStream?: MediaStream;
  displayStream?: MediaStream;
  userStreamHasVideo: boolean;
  displayStreamHasVideo: boolean;
  userMediaStreamID: string;
  uid: string;
};

const peersData = ref<PeerData[]>([]);

function handleStream(stream: MediaStream, uid: string) {
  const i = peersData.value.findIndex((p) => p.uid === uid);
  if (i !== -1) {
    if (stream.id === peersData.value[i].userMediaStreamID)
      peersData.value[i].userStream = stream;
    else peersData.value[i].displayStream = stream;
  }
}

function addPeer(callerId: string) {
  const peer = new Peer({
    initiator: false,
    trickle: false,
    streams: [
      ...(userStream.value ? [userStream.value] : []),
      ...(displayStream.value ? [displayStream.value] : []),
    ],
    iceCompleteTimeout: 2000, // 5 seconds is too long
  });
  peer.on("signal", (signal) => {
    socketStore.send({
      event_type: "CHANNEL_WEBRTC_RETURNING_SIGNAL",
      data: {
        signal: JSON.stringify(signal),
        caller_id: callerId,
        um_stream_id: userMediaStreamID.value,
        um_vid: mediaOptions.value.userMedia.video,
        dm_vid: mediaOptions.value.displayMedia.video,
      },
    } as ChannelWebRTCReturningSignal);
  });
  peer.on("stream", (stream) => handleStream(stream, callerId));
  return peer;
}

function createPeer(uid: string) {
  const peer = new Peer({
    initiator: true,
    trickle: false,
    streams: [
      ...(userStream.value ? [userStream.value] : []),
      ...(displayStream.value ? [displayStream.value] : []),
    ],
    iceCompleteTimeout: 2000, // 5 seconds is too long
  });
  peer.on("signal", (signal) => {
    socketStore.send({
      event_type: "CHANNEL_WEBRTC_SENDING_SIGNAL",
      data: {
        signal: JSON.stringify(signal),
        to_uid: uid,
        um_stream_id: userMediaStreamID.value,
        um_vid: mediaOptions.value.userMedia.video,
        dm_vid: mediaOptions.value.displayMedia.video,
      },
    } as ChannelWebRTCSendingSignal);
  });
  peer.on("stream", (stream) => handleStream(stream, uid));
  return peer;
}

const handleMessage = async (e: MessageEvent) => {
  const msg = JSON.parse(e.data);
  if (!msg) return;
  if (isChannelWebRTCAllUsers(msg)) {
    const peers: PeerData[] = [];
    if (msg.data.users) {
      msg.data.users.forEach((user) => {
        peers.push({
          peer: createPeer(user.uid),
          uid: user.uid,
          userMediaStreamID: user.um_stream_id,
          userStreamHasVideo: user.um_vid,
          displayStreamHasVideo: user.dm_vid,
        });
      });
    }
    peersData.value = peers;
  }
  if (isChannelWebRTCReceivingReturnedSignal(msg)) {
    const peer = peersData.value.find((p) => p.uid === msg.data.uid)?.peer;
    if (peer) {
      await nextTick(() => {
        peer.signal(msg.data.signal);
      });
    } else {
      console.warn("Peer not found");
    }
  }
  if (isChannelWebRTCUserJoined(msg)) {
    const i = peersData.value.findIndex((p) => p.uid === msg.data.caller_id);
    if (i !== -1) {
      peersData.value[i].peer.destroy();
      peersData.value.splice(i, 1);
    }
    const peer = addPeer(msg.data.caller_id);
    peersData.value.push({
      peer: peer,
      userMediaStreamID: msg.data.um_stream_id,
      userStreamHasVideo: msg.data.um_vid,
      displayStreamHasVideo: msg.data.dm_vid,
      uid: msg.data.caller_id,
    });
    await nextTick(() => {
      peer.signal(msg.data.signal);
    });
  }
  if (isChannelWebRTCUserLeft(msg)) {
    const i = peersData.value.findIndex((p) => p.uid === msg.data.uid);
    if (i !== -1) {
      peersData.value[i].peer.destroy();
      peersData.value.splice(i, 1);
    }
  }
  if (isUpdateMediaOptions(msg)) {
    const i = peersData.value.findIndex((p) => p.uid === msg.data.uid);
    if (i !== -1) {
      peersData.value[i].displayStreamHasVideo = msg.data.dm_vid;
      peersData.value[i].userStreamHasVideo = msg.data.um_vid;
      peersData.value[i].userMediaStreamID = msg.data.um_stream_id;
    }
  }
};

onMounted(() => {
  socketStore.socket?.addEventListener("message", handleMessage);
});

onBeforeUnmount(() => {
  socketStore.socket?.removeEventListener("message", handleMessage);
  socketStore.send({
    event_type: "CHANNEL_WEBRTC_LEAVE",
    data: {
      channel_id: roomChannelStore.current,
    },
  } as ChannelWebRTCLeave);
  peersData.value.forEach((p) => p.peer.destroy());
});
</script>

<template>
  <div class="container">
    <div class="vid-chat-users">
      <VidChatUser
        :userMedia="userStream"
        :displayMedia="displayStream"
        :userMediaStreamID="userMediaStreamID"
        :hasDisplayMediaVideo="mediaOptions.displayMedia.video"
        :hasUserMediaVideo="mediaOptions.userMedia.video"
        :isOwner="true"
        :uid="authStore.uid"
        :smaller="true"
      />
      <VidChatUser
        :userMedia="peerData.userStream"
        :displayMedia="peerData.displayStream"
        :userMediaStreamID="peerData.userMediaStreamID"
        :hasDisplayMediaVideo="peerData.displayStreamHasVideo"
        :hasUserMediaVideo="peerData.userStreamHasVideo"
        :isOwner="false"
        :uid="peerData.uid"
        :key="peerData.uid"
        :smaller="true"
        v-for="peerData in peersData"
      />
    </div>
    <VidChatControls
      :updateOptions="updateOptions"
      :exitButtonClicked="exitButtonClicked"
    />
  </div>
</template>

<style lang="scss" scoped>
.container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--gap-md);
  box-sizing: border-box;
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
