import { nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import { IMediaOptions } from "../interfaces/GeneralInterfaces";
import {
  ChannelWebRTCUpdateMediaOptions,
  CallWebRTCUpdateMediaOptions,
} from "../socketHandling/OutEvents";
import {
  selectedAudioInputDevice,
  selectedVideoInputDevice,
  mediaOptions,
} from "../store/DeviceSettingsStore";
import useSocketStore from "../store/SocketStore";

export const useChatMedia = (
  negotiateConnection: Function,
  channelId?: string
) => {
  const socketStore = useSocketStore();

  const userStream = ref<MediaStream | undefined>();
  const displayStream = ref<MediaStream | undefined>();
  const userMediaStreamID = ref("");

  const updateOptions = (opts: Partial<IMediaOptions>) => {
    mediaOptions.value = { ...mediaOptions.value, ...opts };
    if (opts.userMedia) {
      if (opts.userMedia.audio !== undefined) {
        if (userStream.value) {
          const track = userStream.value.getAudioTracks()[0];
          if (track) track.enabled = opts.userMedia.audio;
        } else {
          retrieveMedia();
          return;
        }
      }
      if (opts.userMedia.video !== undefined) {
        if (userStream.value) {
          const track = userStream.value.getVideoTracks()[0];
          if (track) track.enabled = opts.userMedia.video;
        } else {
          retrieveMedia();
          return;
        }
      }
    }
    if (opts.displayMedia) {
      if (opts.displayMedia.video !== undefined) {
        if (displayStream.value) {
          const track = displayStream.value.getVideoTracks()[0];
          if (track) track.enabled = opts.displayMedia.video;
        } else {
          retrieveMedia();
          return;
        }
      }
    }
    socketStore.send(
      channelId
        ? ({
            event_type: "CHANNEL_WEBRTC_UPDATE_MEDIA_OPTIONS",
            data: {
              um_vid: mediaOptions.value.userMedia.video,
              dm_vid: mediaOptions.value.displayMedia.video,
              um_stream_id: userStream.value?.id || "",
              channel_id: channelId,
            },
          } as ChannelWebRTCUpdateMediaOptions)
        : ({
            event_type: "CALL_UPDATE_MEDIA_OPTIONS",
            data: {
              um_vid: mediaOptions.value.userMedia.video,
              dm_vid: mediaOptions.value.displayMedia.video,
              um_stream_id: userStream.value?.id || "",
            },
          } as CallWebRTCUpdateMediaOptions)
    );
  };

  // retrieveMedia is only called when media devices change, and onMounted
  async function retrieveMedia() {
    let userMediaStream: MediaStream | undefined;
    let displayMediaStream: MediaStream | undefined;
    try {
      userMediaStream = await navigator.mediaDevices.getUserMedia({
        audio: mediaOptions.value.userMedia.audio
          ? {
              noiseSuppression: true,
              echoCancellation: true,
              ...(selectedAudioInputDevice.value
                ? { deviceId: { exact: selectedAudioInputDevice.value } }
                : {}),
            }
          : false,
        video: selectedVideoInputDevice.value
          ? { deviceId: { exact: selectedVideoInputDevice.value } }
          : true,
      });
      const vidTrack = userMediaStream.getVideoTracks()[0];
      const sndTrack = userMediaStream.getAudioTracks()[0];
      if (!mediaOptions.value.userMedia.video) {
        if (vidTrack !== undefined) {
          vidTrack.enabled = false;
        }
      } else {
        vidTrack.contentHint = "motion";
        userMediaStreamID.value = userMediaStream.id;
      }
      if (sndTrack) {
        sndTrack.contentHint = "speech";
      }
    } catch (e) {
      userMediaStreamID.value = "FAILED";
    }
    if (mediaOptions.value.displayMedia.video) {
      try {
        displayMediaStream = await navigator.mediaDevices.getDisplayMedia({
          audio: false,
          // has to be true or it throws an error.
          video: true,
        });
        const vidTrack = displayMediaStream.getVideoTracks()[0];
        if (vidTrack) {
          vidTrack.contentHint = "detail";
        }
        const sndTrack = displayMediaStream.getAudioTracks()[0];
        if (sndTrack) {
          sndTrack.contentHint = "music";
        }
      } catch (e) {
        console.warn(e);
      }
    }
    userStream.value = userMediaStream;
    displayStream.value = displayMediaStream;
    userMediaStreamID.value = userMediaStream?.id as string;
    await nextTick(() => negotiateConnection());
  }

  onMounted(retrieveMedia);

  onBeforeUnmount(() => {
    if (userStream.value) {
      userStream.value
        .getTracks()
        .forEach((track) => userStream.value?.removeTrack(track));
    }
    if (displayStream.value) {
      displayStream.value
        .getTracks()
        .forEach((track) => displayStream.value?.removeTrack(track));
    }
    mediaOptions.value = {
      userMedia: { video: false, audio: true },
      displayMedia: { video: false },
    };
  });

  return {
    userStream,
    displayStream,
    userMediaStreamID,
    updateOptions,
  };
};
