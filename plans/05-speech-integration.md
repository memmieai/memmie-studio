# Speech Integration - "Ramble" Feature Design

## Overview

The "Ramble" feature enables users to quickly capture thoughts through voice across all platforms. Speech is transcribed, converted to blobs, and optionally processed by providers - making it the fastest way to get ideas into the system.

## User Experience

### Quick Capture Flow
1. **Tap/Click/Gesture** the Ramble button (or use hotkey)
2. **Speak** naturally - no commands needed
3. **Auto-transcribe** using Whisper API
4. **Create blob** with transcribed text
5. **Optional processing** by active provider

### Platform-Specific Triggers
- **Web**: Click button or press `Space` (hold to record)
- **Mobile**: Tap and hold button, or shake device
- **AR**: Hand gesture (pinch) or voice command "Hey Memmie"
- **CLI**: `memmie ramble` command

## Technical Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Client Device                         │
│  • Capture audio via native APIs                        │
│  • Stream or batch upload                               │
│  • Visual feedback during recording                     │
└─────────────────────────────────────────────────────────┘
                            │
                    Audio Stream/File
                            ▼
┌─────────────────────────────────────────────────────────┐
│              Studio Service (WebSocket)                  │
│  • Receive audio chunks                                 │
│  • Queue for processing                                 │
│  • Return transcription                                 │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                Media Service (8009)                      │
│  • Whisper API transcription                            │
│  • Audio preprocessing                                  │
│  • Language detection                                   │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                 State Service (8006)                     │
│  • Create blob from transcription                       │
│  • Tag with "ramble" metadata                          │
│  • Trigger provider processing                          │
└─────────────────────────────────────────────────────────┘
```

## Implementation

### Web Client - Audio Capture
```typescript
// hooks/useRamble.ts
import { useState, useRef, useCallback } from 'react';
import { useWebSocket } from './useWebSocket';

export const useRamble = (options?: RambleOptions) => {
  const [isRecording, setIsRecording] = useState(false);
  const [transcription, setTranscription] = useState('');
  const mediaRecorder = useRef<MediaRecorder | null>(null);
  const audioChunks = useRef<Blob[]>([]);
  const { send, subscribe } = useWebSocket();
  
  const startRecording = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ 
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          sampleRate: 16000,  // Optimal for Whisper
        } 
      });
      
      mediaRecorder.current = new MediaRecorder(stream, {
        mimeType: 'audio/webm;codecs=opus'
      });
      
      mediaRecorder.current.ondataavailable = (event) => {
        if (event.data.size > 0) {
          audioChunks.current.push(event.data);
          
          // Stream chunks to server for real-time processing
          if (options?.streaming) {
            send({
              type: 'ramble_chunk',
              payload: {
                audio: event.data,
                sequence: audioChunks.current.length
              }
            });
          }
        }
      };
      
      mediaRecorder.current.onstop = async () => {
        if (!options?.streaming) {
          // Send complete audio for batch processing
          const audioBlob = new Blob(audioChunks.current, { 
            type: 'audio/webm' 
          });
          
          send({
            type: 'ramble_complete',
            payload: {
              audio: audioBlob,
              provider_id: options?.providerId,
              metadata: options?.metadata
            }
          });
        }
        
        audioChunks.current = [];
      };
      
      mediaRecorder.current.start(options?.chunkSize || 1000); // 1 second chunks
      setIsRecording(true);
      
      // Visual feedback
      if (options?.onStart) options.onStart();
      
    } catch (error) {
      console.error('Failed to start recording:', error);
      if (options?.onError) options.onError(error);
    }
  }, [options, send]);
  
  const stopRecording = useCallback(() => {
    if (mediaRecorder.current && isRecording) {
      mediaRecorder.current.stop();
      mediaRecorder.current.stream.getTracks().forEach(track => track.stop());
      setIsRecording(false);
      
      if (options?.onStop) options.onStop();
    }
  }, [isRecording, options]);
  
  // Subscribe to transcription results
  useEffect(() => {
    const unsubscribe = subscribe('ramble_result', (data) => {
      setTranscription(data.transcription);
      if (options?.onTranscription) {
        options.onTranscription(data);
      }
    });
    
    return unsubscribe;
  }, [subscribe, options]);
  
  return {
    isRecording,
    transcription,
    startRecording,
    stopRecording,
  };
};
```

### Ramble Button Component
```typescript
// components/RambleButton.tsx
import React, { useState, useEffect } from 'react';
import { useRamble } from '@/hooks/useRamble';
import { motion, AnimatePresence } from 'framer-motion';
import { Mic, MicOff, Loader } from 'lucide-react';

interface RambleButtonProps {
  providerId?: string;
  floating?: boolean;
  position?: 'bottom-right' | 'bottom-center' | 'top-right';
  maxDuration?: number;  // seconds
  autoTranscribe?: boolean;
  onComplete?: (blob: Blob) => void;
}

export const RambleButton: React.FC<RambleButtonProps> = ({
  providerId,
  floating = true,
  position = 'bottom-right',
  maxDuration = 300,
  autoTranscribe = true,
  onComplete,
}) => {
  const [timeRemaining, setTimeRemaining] = useState(maxDuration);
  const [isProcessing, setIsProcessing] = useState(false);
  
  const {
    isRecording,
    transcription,
    startRecording,
    stopRecording,
  } = useRamble({
    providerId,
    streaming: true,
    onTranscription: (data) => {
      setIsProcessing(false);
      if (onComplete) onComplete(data.blob);
    },
    onStop: () => {
      setIsProcessing(true);
    }
  });
  
  // Auto-stop after max duration
  useEffect(() => {
    if (isRecording) {
      const timer = setInterval(() => {
        setTimeRemaining(prev => {
          if (prev <= 1) {
            stopRecording();
            return maxDuration;
          }
          return prev - 1;
        });
      }, 1000);
      
      return () => clearInterval(timer);
    } else {
      setTimeRemaining(maxDuration);
    }
  }, [isRecording, maxDuration, stopRecording]);
  
  // Keyboard shortcut
  useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      if (e.code === 'Space' && !e.repeat && 
          !['INPUT', 'TEXTAREA'].includes(document.activeElement?.tagName || '')) {
        e.preventDefault();
        if (!isRecording) {
          startRecording();
        }
      }
    };
    
    const handleKeyUp = (e: KeyboardEvent) => {
      if (e.code === 'Space' && isRecording) {
        e.preventDefault();
        stopRecording();
      }
    };
    
    window.addEventListener('keydown', handleKeyPress);
    window.addEventListener('keyup', handleKeyUp);
    
    return () => {
      window.removeEventListener('keydown', handleKeyPress);
      window.removeEventListener('keyup', handleKeyUp);
    };
  }, [isRecording, startRecording, stopRecording]);
  
  const positionClasses = {
    'bottom-right': 'bottom-6 right-6',
    'bottom-center': 'bottom-6 left-1/2 -translate-x-1/2',
    'top-right': 'top-6 right-6',
  };
  
  return (
    <motion.div
      className={`
        ${floating ? 'fixed' : 'relative'}
        ${positionClasses[position]}
        z-50
      `}
      initial={{ scale: 0 }}
      animate={{ scale: 1 }}
      exit={{ scale: 0 }}
    >
      <motion.button
        className={`
          relative rounded-full p-6 shadow-2xl
          ${isRecording ? 'bg-red-500' : 'bg-blue-500'}
          ${isProcessing ? 'bg-gray-400' : ''}
          text-white transition-colors
        `}
        whileTap={{ scale: 0.95 }}
        onClick={isRecording ? stopRecording : startRecording}
        disabled={isProcessing}
      >
        {/* Pulse animation when recording */}
        {isRecording && (
          <motion.div
            className="absolute inset-0 rounded-full bg-red-500"
            animate={{
              scale: [1, 1.3, 1],
              opacity: [0.5, 0, 0.5],
            }}
            transition={{
              duration: 1.5,
              repeat: Infinity,
            }}
          />
        )}
        
        {/* Icon */}
        <div className="relative z-10">
          {isProcessing ? (
            <Loader className="w-8 h-8 animate-spin" />
          ) : isRecording ? (
            <MicOff className="w-8 h-8" />
          ) : (
            <Mic className="w-8 h-8" />
          )}
        </div>
        
        {/* Timer */}
        {isRecording && (
          <div className="absolute -top-10 left-1/2 -translate-x-1/2 
                          bg-black/75 text-white px-2 py-1 rounded text-sm">
            {Math.floor(timeRemaining / 60)}:{(timeRemaining % 60).toString().padStart(2, '0')}
          </div>
        )}
      </motion.button>
      
      {/* Transcription preview */}
      <AnimatePresence>
        {transcription && (
          <motion.div
            className="absolute bottom-full mb-4 right-0 
                       bg-white rounded-lg shadow-lg p-4 
                       max-w-sm max-h-32 overflow-y-auto"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 10 }}
          >
            <p className="text-sm text-gray-700">{transcription}</p>
          </motion.div>
        )}
      </AnimatePresence>
      
      {/* Instructions */}
      {!isRecording && !isProcessing && (
        <div className="absolute -bottom-12 left-1/2 -translate-x-1/2 
                        text-xs text-gray-500 whitespace-nowrap">
          Click or hold Space to record
        </div>
      )}
    </motion.div>
  );
};
```

### Mobile Implementation (React Native)
```typescript
// components/native/RambleButton.tsx
import React, { useState, useEffect } from 'react';
import {
  TouchableOpacity,
  View,
  Text,
  Animated,
  Vibration,
  Platform,
} from 'react-native';
import AudioRecorderPlayer from 'react-native-audio-recorder-player';
import { Permissions } from 'react-native-permissions';
import Icon from 'react-native-vector-icons/MaterialIcons';

const audioRecorderPlayer = new AudioRecorderPlayer();

export const RambleButton: React.FC<RambleButtonProps> = ({
  onComplete,
  maxDuration = 300,
}) => {
  const [isRecording, setIsRecording] = useState(false);
  const [recordTime, setRecordTime] = useState('00:00');
  const pulseAnim = useRef(new Animated.Value(1)).current;
  
  const startRecording = async () => {
    // Request permissions
    if (Platform.OS === 'android') {
      const grants = await Permissions.requestMultiple([
        Permissions.PERMISSIONS.ANDROID.RECORD_AUDIO,
        Permissions.PERMISSIONS.ANDROID.WRITE_EXTERNAL_STORAGE,
      ]);
    } else {
      await Permissions.request(Permissions.PERMISSIONS.IOS.MICROPHONE);
    }
    
    // Start recording
    const path = Platform.select({
      ios: 'ramble.m4a',
      android: `${RNFS.ExternalDirectoryPath}/ramble.mp4`,
    });
    
    const result = await audioRecorderPlayer.startRecorder(path);
    audioRecorderPlayer.addRecordBackListener((e) => {
      setRecordTime(audioRecorderPlayer.mmssss(Math.floor(e.currentPosition)));
      
      // Auto-stop at max duration
      if (e.currentPosition >= maxDuration * 1000) {
        stopRecording();
      }
    });
    
    setIsRecording(true);
    Vibration.vibrate(100);  // Haptic feedback
    
    // Start pulse animation
    Animated.loop(
      Animated.sequence([
        Animated.timing(pulseAnim, {
          toValue: 1.2,
          duration: 500,
          useNativeDriver: true,
        }),
        Animated.timing(pulseAnim, {
          toValue: 1,
          duration: 500,
          useNativeDriver: true,
        }),
      ])
    ).start();
  };
  
  const stopRecording = async () => {
    const result = await audioRecorderPlayer.stopRecorder();
    audioRecorderPlayer.removeRecordBackListener();
    setIsRecording(false);
    setRecordTime('00:00');
    Vibration.vibrate(100);
    
    // Upload audio for transcription
    uploadAudio(result);
  };
  
  const uploadAudio = async (audioPath: string) => {
    const formData = new FormData();
    formData.append('audio', {
      uri: audioPath,
      type: 'audio/mp4',
      name: 'ramble.mp4',
    } as any);
    
    const response = await fetch(`${API_URL}/api/v1/ramble`, {
      method: 'POST',
      body: formData,
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    
    const result = await response.json();
    onComplete?.(result);
  };
  
  return (
    <View style={styles.container}>
      <TouchableOpacity
        onPress={isRecording ? stopRecording : startRecording}
        activeOpacity={0.8}
      >
        <Animated.View
          style={[
            styles.button,
            isRecording && styles.recordingButton,
            { transform: [{ scale: pulseAnim }] },
          ]}
        >
          <Icon
            name={isRecording ? 'stop' : 'mic'}
            size={32}
            color="white"
          />
        </Animated.View>
      </TouchableOpacity>
      
      {isRecording && (
        <Text style={styles.timer}>{recordTime}</Text>
      )}
    </View>
  );
};
```

### AR Implementation (Vision Pro)
```swift
// RambleButton.swift
import SwiftUI
import RealityKit
import Speech

struct RambleButton3D: View {
    @State private var isRecording = false
    @State private var transcription = ""
    @StateObject private var speechRecognizer = SpeechRecognizer()
    
    var body: some View {
        ZStack {
            // 3D microphone model
            Model3D(named: "microphone") { model in
                model
                    .resizable()
                    .scaledToFit()
                    .frame(width: 100, height: 100)
                    .overlay(
                        // Pulsing effect when recording
                        isRecording ? PulsingOverlay() : nil
                    )
            } placeholder: {
                ProgressView()
            }
            .onTapGesture {
                toggleRecording()
            }
            // Voice activation
            .onVoiceCommand("start recording") {
                startRecording()
            }
            .onVoiceCommand("stop recording") {
                stopRecording()
            }
            
            // Floating transcription bubble
            if !transcription.isEmpty {
                Text(transcription)
                    .padding()
                    .background(.ultraThinMaterial)
                    .cornerRadius(20)
                    .offset(y: -150)
                    .transition(.scale.combined(with: .opacity))
            }
        }
        // Hand gesture recognition
        .onHandGesture(.pinch) { gesture in
            if gesture.state == .began {
                startRecording()
            } else if gesture.state == .ended {
                stopRecording()
            }
        }
    }
    
    private func startRecording() {
        speechRecognizer.startTranscribing { text in
            transcription = text
        }
        isRecording = true
        
        // Haptic feedback
        UIImpactFeedbackGenerator(style: .medium).impactOccurred()
    }
    
    private func stopRecording() {
        speechRecognizer.stopTranscribing()
        isRecording = false
        
        // Send to server
        Task {
            await createBlobFromTranscription(transcription)
        }
    }
}
```

## Server-Side Processing

### Studio Service WebSocket Handler
```go
func (s *StudioService) HandleRambleWebSocket(ws *websocket.Conn) {
    audioBuffer := &bytes.Buffer{}
    
    for {
        var msg RambleMessage
        if err := ws.ReadJSON(&msg); err != nil {
            break
        }
        
        switch msg.Type {
        case "ramble_start":
            audioBuffer.Reset()
            ws.WriteJSON(map[string]string{
                "type": "ramble_started",
                "status": "recording",
            })
            
        case "ramble_chunk":
            // Accumulate audio chunks
            audioData, _ := base64.StdEncoding.DecodeString(msg.Payload.Audio)
            audioBuffer.Write(audioData)
            
            // Optional: Stream to Whisper for real-time transcription
            if msg.Payload.Streaming {
                go s.streamTranscribe(audioBuffer.Bytes(), ws)
            }
            
        case "ramble_complete":
            // Process complete audio
            transcription, err := s.transcribeAudio(audioBuffer.Bytes())
            if err != nil {
                ws.WriteJSON(map[string]string{
                    "type": "error",
                    "message": err.Error(),
                })
                continue
            }
            
            // Create blob
            blob, err := s.createBlobFromTranscription(
                msg.Payload.UserID,
                transcription,
                msg.Payload.ProviderID,
            )
            
            ws.WriteJSON(map[string]interface{}{
                "type": "ramble_result",
                "transcription": transcription,
                "blob": blob,
            })
        }
    }
}

func (s *StudioService) transcribeAudio(audio []byte) (string, error) {
    // Call Media Service for Whisper transcription
    resp, err := s.mediaClient.Transcribe(context.Background(), &media.TranscribeRequest{
        Audio:     audio,
        Model:     "whisper-1",
        Language:  "auto",  // Auto-detect language
        Prompt:    "",      // Optional context
    })
    
    if err != nil {
        return "", err
    }
    
    return resp.Text, nil
}
```

### Media Service Whisper Integration
```go
package media

import (
    "github.com/sashabaranov/go-openai"
)

type WhisperService struct {
    client *openai.Client
}

func (s *WhisperService) Transcribe(ctx context.Context, req *TranscribeRequest) (*TranscribeResponse, error) {
    // Prepare audio file
    audioFile := bytes.NewReader(req.Audio)
    
    // Call OpenAI Whisper API
    resp, err := s.client.CreateTranscription(ctx, openai.AudioRequest{
        Model:    openai.Whisper1,
        FilePath: "audio.webm",
        File:     audioFile,
        Language: req.Language,
        Prompt:   req.Prompt,
    })
    
    if err != nil {
        return nil, err
    }
    
    return &TranscribeResponse{
        Text:     resp.Text,
        Language: resp.Language,
        Duration: resp.Duration,
        Segments: resp.Segments,
    }, nil
}
```

## Advanced Features

### Context-Aware Transcription
```go
// Provide context to improve transcription accuracy
func (s *WhisperService) TranscribeWithContext(ctx context.Context, audio []byte, context string) (string, error) {
    // Get recent blobs for context
    recentBlobs, _ := s.stateClient.GetRecentBlobs(ctx, userID, 5)
    
    // Build context prompt
    contextPrompt := fmt.Sprintf(
        "Previous content: %s. Current topic: %s",
        extractText(recentBlobs),
        context,
    )
    
    resp, err := s.client.CreateTranscription(ctx, openai.AudioRequest{
        Model:  openai.Whisper1,
        File:   audio,
        Prompt: contextPrompt,  // Improves accuracy
    })
    
    return resp.Text, err
}
```

### Multi-Language Support
```typescript
const languages = {
  en: 'English',
  es: 'Spanish',
  fr: 'French',
  de: 'German',
  ja: 'Japanese',
  zh: 'Chinese',
  // ... 50+ languages supported by Whisper
};

// Auto-detect or let user select
const { language, autoDetect } = useLanguageSettings();
```

### Voice Commands
```typescript
// Voice command system
const voiceCommands = {
  'new chapter': () => createBlob({ type: 'chapter' }),
  'expand this': () => executeProvider('text-expander'),
  'save draft': () => saveDraft(),
  'read back': () => textToSpeech(currentBlob),
};

// Process transcription for commands
function processVoiceCommand(transcription: string) {
  const command = transcription.toLowerCase().trim();
  
  for (const [trigger, action] of Object.entries(voiceCommands)) {
    if (command.includes(trigger)) {
      action();
      return true;
    }
  }
  
  return false;
}
```

## Accessibility

### Screen Reader Support
```typescript
<button
  aria-label={isRecording ? "Stop recording" : "Start voice recording"}
  aria-pressed={isRecording}
  aria-describedby="ramble-instructions"
>
  {/* ... */}
</button>
<span id="ramble-instructions" className="sr-only">
  Press and hold Space bar, or click this button to record your voice
</span>
```

### Visual Indicators
- Pulsing animation during recording
- Timer showing remaining time
- Transcription preview
- Processing spinner

### Haptic Feedback
- Vibration on start/stop (mobile)
- Force feedback (AR devices)

## Performance Optimizations

### Audio Compression
```typescript
// Compress audio before upload
async function compressAudio(blob: Blob): Promise<Blob> {
  const audioContext = new AudioContext();
  const arrayBuffer = await blob.arrayBuffer();
  const audioBuffer = await audioContext.decodeAudioData(arrayBuffer);
  
  // Downsample to 16kHz (optimal for speech)
  const offlineContext = new OfflineAudioContext(
    1,  // Mono
    audioBuffer.duration * 16000,
    16000  // 16kHz sample rate
  );
  
  const source = offlineContext.createBufferSource();
  source.buffer = audioBuffer;
  source.connect(offlineContext.destination);
  source.start();
  
  const renderedBuffer = await offlineContext.startRendering();
  return audioBufferToBlob(renderedBuffer);
}
```

### Streaming Transcription
```go
// Stream audio chunks for real-time transcription
func (s *WhisperService) StreamTranscribe(stream io.Reader) (<-chan string, error) {
    transcriptions := make(chan string)
    
    go func() {
        defer close(transcriptions)
        
        buffer := make([]byte, 16000) // 1 second at 16kHz
        for {
            n, err := stream.Read(buffer)
            if err == io.EOF {
                break
            }
            
            // Process chunk
            text, _ := s.transcribeChunk(buffer[:n])
            transcriptions <- text
        }
    }()
    
    return transcriptions, nil
}
```

## Privacy & Security

1. **Encryption**: Audio encrypted in transit (TLS)
2. **Temporary Storage**: Audio deleted after transcription
3. **User Control**: Option to process locally (future)
4. **Consent**: Clear permission requests
5. **Data Minimization**: Only transcription stored, not audio

This comprehensive speech integration enables natural voice input across all platforms, making Memmie Studio the fastest way to capture and process thoughts.