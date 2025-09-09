# UI Implementation Tasks - React Native Web

## Why React Native Web?
- **Single codebase** for Web, iOS, and Android
- **Native performance** on mobile
- **Dynamic UI generation** support via component mapping
- **Consistent design system** across platforms
- **Hot reload** for rapid development
- **TypeScript** for type safety

## Prerequisites
- Node.js 18+
- Yarn or npm
- Expo CLI for React Native development

## Task 1: Initialize React Native Web Project
```bash
cd /home/uneid/iter3/memmieai
npx create-expo-app memmie-studio-ui --template blank-typescript
cd memmie-studio-ui

# Add web support
npx expo install react-native-web react-dom @expo/webpack-config

# Add navigation
yarn add @react-navigation/native @react-navigation/stack @react-navigation/bottom-tabs
yarn add react-native-screens react-native-safe-area-context
yarn add @react-navigation/native-stack

# Add UI components
yarn add react-native-elements react-native-vector-icons
yarn add react-native-paper

# Add state management
yarn add @reduxjs/toolkit react-redux redux-persist
yarn add @tanstack/react-query

# Add WebSocket
yarn add socket.io-client

# Add form handling
yarn add react-hook-form yup

# Add utilities
yarn add axios date-fns lodash
yarn add react-native-async-storage/async-storage
```

## Task 2: Project Structure
```
memmie-studio-ui/
├── src/
│   ├── api/
│   │   ├── auth.ts
│   │   ├── blobs.ts
│   │   ├── buckets.ts
│   │   ├── studio.ts
│   │   └── websocket.ts
│   ├── components/
│   │   ├── common/
│   │   │   ├── Button.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Card.tsx
│   │   │   └── Loading.tsx
│   │   ├── buckets/
│   │   │   ├── BucketTree.tsx
│   │   │   ├── BucketCard.tsx
│   │   │   └── BucketCreator.tsx
│   │   ├── blobs/
│   │   │   ├── BlobEditor.tsx
│   │   │   ├── BlobViewer.tsx
│   │   │   └── BlobList.tsx
│   │   ├── books/
│   │   │   ├── BookCreator.tsx
│   │   │   ├── ChapterEditor.tsx
│   │   │   └── BookExporter.tsx
│   │   └── dynamic/
│   │       ├── DynamicComponent.tsx
│   │       ├── ComponentRegistry.tsx
│   │       └── SchemaRenderer.tsx
│   ├── screens/
│   │   ├── auth/
│   │   │   ├── LoginScreen.tsx
│   │   │   └── RegisterScreen.tsx
│   │   ├── home/
│   │   │   └── HomeScreen.tsx
│   │   ├── books/
│   │   │   ├── BooksListScreen.tsx
│   │   │   ├── BookDetailScreen.tsx
│   │   │   └── ChapterEditScreen.tsx
│   │   └── settings/
│   │       └── SettingsScreen.tsx
│   ├── navigation/
│   │   ├── AppNavigator.tsx
│   │   ├── AuthNavigator.tsx
│   │   └── TabNavigator.tsx
│   ├── store/
│   │   ├── index.ts
│   │   ├── authSlice.ts
│   │   ├── blobSlice.ts
│   │   ├── bucketSlice.ts
│   │   └── websocketSlice.ts
│   ├── hooks/
│   │   ├── useWebSocket.ts
│   │   ├── useAuth.ts
│   │   ├── useBlobs.ts
│   │   └── useBuckets.ts
│   ├── utils/
│   │   ├── storage.ts
│   │   ├── validation.ts
│   │   └── formatting.ts
│   ├── theme/
│   │   ├── colors.ts
│   │   ├── typography.ts
│   │   └── spacing.ts
│   └── types/
│       ├── api.ts
│       ├── models.ts
│       └── navigation.ts
├── App.tsx
├── app.json
├── babel.config.js
├── tsconfig.json
├── webpack.config.js
└── package.json
```

## Task 3: Configure TypeScript and Babel
**File**: `tsconfig.json`
```json
{
  "extends": "expo/tsconfig.base",
  "compilerOptions": {
    "strict": true,
    "jsx": "react-native",
    "lib": ["ES2022"],
    "moduleResolution": "node",
    "allowSyntheticDefaultImports": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "resolveJsonModule": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"],
      "@components/*": ["src/components/*"],
      "@screens/*": ["src/screens/*"],
      "@api/*": ["src/api/*"],
      "@hooks/*": ["src/hooks/*"],
      "@utils/*": ["src/utils/*"],
      "@store/*": ["src/store/*"],
      "@types/*": ["src/types/*"]
    }
  }
}
```

## Task 4: Define Type Definitions
**File**: `src/types/models.ts`
```typescript
export interface User {
  id: string;
  email: string;
  username: string;
  phone?: string;
}

export interface Blob {
  id: string;
  userId: string;
  processorId: string;
  schemaId: string;
  data: any;
  bucketIds: string[];
  parentId?: string;
  derivedIds: string[];
  title?: string;
  preview?: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface Bucket {
  id: string;
  userId: string;
  name: string;
  type: string;
  parentBucketId?: string;
  childBucketIds: string[];
  blobIds: string[];
  metadata: Record<string, any>;
  description?: string;
  icon?: string;
  color?: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface Book extends Bucket {
  type: 'book';
  metadata: {
    author: string;
    genre: string;
    chaptersPlanned: number;
    wordCount?: number;
  };
}

export interface Chapter extends Bucket {
  type: 'chapter';
  metadata: {
    chapterNumber: number;
    title: string;
    status: 'draft' | 'review' | 'complete';
  };
}

export interface WebSocketMessage {
  id?: string;
  type: string;
  data: any;
  timestamp: number;
}

export interface Schema {
  id: string;
  name: string;
  version: string;
  definition: object;
  uiSchema?: object; // For dynamic UI generation
}
```

## Task 5: Implement API Client
**File**: `src/api/client.ts`
```typescript
import axios, { AxiosInstance } from 'axios';
import AsyncStorage from '@react-native-async-storage/async-storage';

const API_BASE_URL = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8000';

class ApiClient {
  private instance: AxiosInstance;

  constructor() {
    this.instance = axios.create({
      baseURL: API_BASE_URL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor to add auth token
    this.instance.interceptors.request.use(
      async (config) => {
        const token = await AsyncStorage.getItem('authToken');
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor for error handling
    this.instance.interceptors.response.use(
      (response) => response,
      async (error) => {
        if (error.response?.status === 401) {
          // Token expired, redirect to login
          await AsyncStorage.removeItem('authToken');
          // Dispatch logout action
        }
        return Promise.reject(error);
      }
    );
  }

  get<T>(url: string, params?: any) {
    return this.instance.get<T>(url, { params });
  }

  post<T>(url: string, data?: any) {
    return this.instance.post<T>(url, data);
  }

  put<T>(url: string, data?: any) {
    return this.instance.put<T>(url, data);
  }

  delete<T>(url: string) {
    return this.instance.delete<T>(url);
  }
}

export default new ApiClient();
```

## Task 6: Implement WebSocket Hook
**File**: `src/hooks/useWebSocket.ts`
```typescript
import { useEffect, useRef, useCallback } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import io, { Socket } from 'socket.io-client';
import { RootState } from '@/store';
import { 
  setConnected, 
  addMessage, 
  setError 
} from '@/store/websocketSlice';

const WS_URL = process.env.EXPO_PUBLIC_WS_URL || 'ws://localhost:8000';

export const useWebSocket = () => {
  const dispatch = useDispatch();
  const socketRef = useRef<Socket | null>(null);
  const { isAuthenticated, token } = useSelector((state: RootState) => state.auth);

  const connect = useCallback(() => {
    if (!isAuthenticated || !token) return;

    socketRef.current = io(WS_URL, {
      transports: ['websocket'],
      auth: { token },
      reconnection: true,
      reconnectionDelay: 1000,
      reconnectionAttempts: 5,
    });

    socketRef.current.on('connect', () => {
      console.log('WebSocket connected');
      dispatch(setConnected(true));
    });

    socketRef.current.on('disconnect', () => {
      console.log('WebSocket disconnected');
      dispatch(setConnected(false));
    });

    socketRef.current.on('blob.created', (data) => {
      dispatch(addMessage({ type: 'blob.created', data }));
    });

    socketRef.current.on('blob.derived', (data) => {
      dispatch(addMessage({ type: 'blob.derived', data }));
    });

    socketRef.current.on('error', (error) => {
      dispatch(setError(error.message));
    });
  }, [isAuthenticated, token, dispatch]);

  const disconnect = useCallback(() => {
    if (socketRef.current) {
      socketRef.current.disconnect();
      socketRef.current = null;
    }
  }, []);

  const emit = useCallback((event: string, data: any) => {
    if (socketRef.current?.connected) {
      socketRef.current.emit(event, data);
    }
  }, []);

  const subscribe = useCallback((bucketIds: string[]) => {
    emit('subscribe', { bucket_ids: bucketIds });
  }, [emit]);

  useEffect(() => {
    if (isAuthenticated) {
      connect();
    } else {
      disconnect();
    }

    return () => {
      disconnect();
    };
  }, [isAuthenticated, connect, disconnect]);

  return {
    emit,
    subscribe,
    disconnect,
  };
};
```

## Task 7: Create Dynamic Component System
**File**: `src/components/dynamic/DynamicComponent.tsx`
```tsx
import React from 'react';
import { View, Text } from 'react-native';
import { ComponentRegistry } from './ComponentRegistry';

interface DynamicComponentProps {
  schema: any;
  data: any;
  onChange?: (data: any) => void;
  readonly?: boolean;
}

export const DynamicComponent: React.FC<DynamicComponentProps> = ({
  schema,
  data,
  onChange,
  readonly = false,
}) => {
  const renderField = (fieldSchema: any, fieldData: any, path: string) => {
    const Component = ComponentRegistry.getComponent(fieldSchema.type, fieldSchema.format);
    
    if (!Component) {
      return (
        <Text>Unknown field type: {fieldSchema.type}</Text>
      );
    }

    return (
      <Component
        key={path}
        schema={fieldSchema}
        value={fieldData}
        onChange={(value: any) => {
          if (onChange && !readonly) {
            const newData = { ...data };
            setNestedValue(newData, path, value);
            onChange(newData);
          }
        }}
        readonly={readonly}
        path={path}
      />
    );
  };

  const renderObject = (objSchema: any, objData: any, basePath: string = '') => {
    if (!objSchema.properties) return null;

    return Object.entries(objSchema.properties).map(([key, fieldSchema]: [string, any]) => {
      const path = basePath ? `${basePath}.${key}` : key;
      const fieldData = objData?.[key];

      if (fieldSchema.type === 'object') {
        return (
          <View key={path}>
            <Text style={styles.sectionTitle}>{fieldSchema.title || key}</Text>
            {renderObject(fieldSchema, fieldData, path)}
          </View>
        );
      }

      if (fieldSchema.type === 'array') {
        return renderArray(fieldSchema, fieldData, path);
      }

      return renderField(fieldSchema, fieldData, path);
    });
  };

  const renderArray = (arraySchema: any, arrayData: any[], path: string) => {
    const items = arrayData || [];
    
    return (
      <View key={path}>
        <Text style={styles.sectionTitle}>{arraySchema.title || path}</Text>
        {items.map((item, index) => (
          <View key={`${path}[${index}]`}>
            {arraySchema.items.type === 'object' 
              ? renderObject(arraySchema.items, item, `${path}[${index}]`)
              : renderField(arraySchema.items, item, `${path}[${index}]`)
            }
          </View>
        ))}
      </View>
    );
  };

  return (
    <View style={styles.container}>
      {schema.type === 'object' 
        ? renderObject(schema, data)
        : renderField(schema, data, 'root')
      }
    </View>
  );
};

const setNestedValue = (obj: any, path: string, value: any) => {
  const keys = path.split(/\.|\[|\]/).filter(Boolean);
  let current = obj;
  
  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i];
    if (!current[key]) {
      current[key] = isNaN(Number(keys[i + 1])) ? {} : [];
    }
    current = current[key];
  }
  
  current[keys[keys.length - 1]] = value;
};

const styles = {
  container: {
    padding: 16,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: 'bold',
    marginTop: 16,
    marginBottom: 8,
  },
};
```

## Task 8: Create Component Registry
**File**: `src/components/dynamic/ComponentRegistry.tsx`
```tsx
import React from 'react';
import { TextInput, Switch, View, Text } from 'react-native';
import { Picker } from '@react-native-picker/picker';
import DateTimePicker from '@react-native-community/datetimepicker';

type ComponentType = React.FC<any>;

class Registry {
  private components: Map<string, ComponentType> = new Map();

  constructor() {
    this.registerDefaultComponents();
  }

  private registerDefaultComponents() {
    // Text components
    this.register('string', null, StringComponent);
    this.register('string', 'email', EmailComponent);
    this.register('string', 'password', PasswordComponent);
    this.register('string', 'multiline', TextAreaComponent);
    
    // Number components
    this.register('number', null, NumberComponent);
    this.register('integer', null, IntegerComponent);
    
    // Boolean
    this.register('boolean', null, BooleanComponent);
    
    // Date/Time
    this.register('string', 'date', DateComponent);
    this.register('string', 'date-time', DateTimeComponent);
    
    // Select/Enum
    this.register('string', 'select', SelectComponent);
  }

  register(type: string, format: string | null, component: ComponentType) {
    const key = format ? `${type}:${format}` : type;
    this.components.set(key, component);
  }

  getComponent(type: string, format?: string): ComponentType | undefined {
    const key = format ? `${type}:${format}` : type;
    return this.components.get(key) || this.components.get(type);
  }
}

// Component implementations
const StringComponent: React.FC<any> = ({ schema, value, onChange, readonly }) => (
  <View style={styles.fieldContainer}>
    <Text style={styles.label}>{schema.title || schema.name}</Text>
    <TextInput
      style={styles.input}
      value={value || ''}
      onChangeText={onChange}
      placeholder={schema.description}
      editable={!readonly}
    />
  </View>
);

const EmailComponent: React.FC<any> = ({ schema, value, onChange, readonly }) => (
  <View style={styles.fieldContainer}>
    <Text style={styles.label}>{schema.title || 'Email'}</Text>
    <TextInput
      style={styles.input}
      value={value || ''}
      onChangeText={onChange}
      placeholder={schema.description}
      keyboardType="email-address"
      autoCapitalize="none"
      editable={!readonly}
    />
  </View>
);

const PasswordComponent: React.FC<any> = ({ schema, value, onChange, readonly }) => (
  <View style={styles.fieldContainer}>
    <Text style={styles.label}>{schema.title || 'Password'}</Text>
    <TextInput
      style={styles.input}
      value={value || ''}
      onChangeText={onChange}
      placeholder={schema.description}
      secureTextEntry
      editable={!readonly}
    />
  </View>
);

const TextAreaComponent: React.FC<any> = ({ schema, value, onChange, readonly }) => (
  <View style={styles.fieldContainer}>
    <Text style={styles.label}>{schema.title || schema.name}</Text>
    <TextInput
      style={[styles.input, styles.textArea]}
      value={value || ''}
      onChangeText={onChange}
      placeholder={schema.description}
      multiline
      numberOfLines={4}
      editable={!readonly}
    />
  </View>
);

const NumberComponent: React.FC<any> = ({ schema, value, onChange, readonly }) => (
  <View style={styles.fieldContainer}>
    <Text style={styles.label}>{schema.title || schema.name}</Text>
    <TextInput
      style={styles.input}
      value={String(value || '')}
      onChangeText={(text) => onChange(parseFloat(text) || 0)}
      placeholder={schema.description}
      keyboardType="numeric"
      editable={!readonly}
    />
  </View>
);

const BooleanComponent: React.FC<any> = ({ schema, value, onChange, readonly }) => (
  <View style={styles.fieldContainer}>
    <Text style={styles.label}>{schema.title || schema.name}</Text>
    <Switch
      value={!!value}
      onValueChange={onChange}
      disabled={readonly}
    />
  </View>
);

const SelectComponent: React.FC<any> = ({ schema, value, onChange, readonly }) => (
  <View style={styles.fieldContainer}>
    <Text style={styles.label}>{schema.title || schema.name}</Text>
    <Picker
      selectedValue={value}
      onValueChange={onChange}
      enabled={!readonly}
      style={styles.picker}
    >
      {schema.enum?.map((option: string) => (
        <Picker.Item key={option} label={option} value={option} />
      ))}
    </Picker>
  </View>
);

const DateComponent: React.FC<any> = ({ schema, value, onChange, readonly }) => {
  const [show, setShow] = React.useState(false);
  const date = value ? new Date(value) : new Date();

  return (
    <View style={styles.fieldContainer}>
      <Text style={styles.label}>{schema.title || 'Date'}</Text>
      <Text 
        style={styles.dateText}
        onPress={() => !readonly && setShow(true)}
      >
        {value ? date.toLocaleDateString() : 'Select date'}
      </Text>
      {show && (
        <DateTimePicker
          value={date}
          mode="date"
          onChange={(event, selectedDate) => {
            setShow(false);
            if (selectedDate) {
              onChange(selectedDate.toISOString());
            }
          }}
        />
      )}
    </View>
  );
};

const styles = {
  fieldContainer: {
    marginBottom: 16,
  },
  label: {
    fontSize: 14,
    fontWeight: '600',
    marginBottom: 4,
    color: '#333',
  },
  input: {
    borderWidth: 1,
    borderColor: '#ddd',
    borderRadius: 8,
    padding: 12,
    fontSize: 16,
  },
  textArea: {
    minHeight: 100,
    textAlignVertical: 'top',
  },
  picker: {
    borderWidth: 1,
    borderColor: '#ddd',
    borderRadius: 8,
  },
  dateText: {
    borderWidth: 1,
    borderColor: '#ddd',
    borderRadius: 8,
    padding: 12,
    fontSize: 16,
  },
};

export const ComponentRegistry = new Registry();
```

## Task 9: Create Book Editor Screen
**File**: `src/screens/books/BookEditScreen.tsx`
```tsx
import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  ScrollView,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  Alert,
} from 'react-native';
import { useNavigation, useRoute } from '@react-navigation/native';
import { useWebSocket } from '@/hooks/useWebSocket';
import { useBlobs } from '@/hooks/useBlobs';
import { useBuckets } from '@/hooks/useBuckets';
import { BucketTree } from '@/components/buckets/BucketTree';
import { BlobEditor } from '@/components/blobs/BlobEditor';

export const BookEditScreen: React.FC = () => {
  const navigation = useNavigation();
  const route = useRoute();
  const { bookId } = route.params as { bookId: string };
  
  const { subscribe } = useWebSocket();
  const { createBlob, updateBlob } = useBlobs();
  const { getBucket, exportBucket } = useBuckets();
  
  const [book, setBook] = useState<any>(null);
  const [selectedChapter, setSelectedChapter] = useState<string | null>(null);
  const [content, setContent] = useState('');
  const [expandedContent, setExpandedContent] = useState('');
  const [isExpanding, setIsExpanding] = useState(false);

  useEffect(() => {
    loadBook();
    subscribe([bookId]);
  }, [bookId]);

  const loadBook = async () => {
    const bookData = await getBucket(bookId);
    setBook(bookData);
  };

  const handleSaveContent = async () => {
    if (!selectedChapter || !content) return;

    try {
      const blob = await createBlob({
        processor_id: 'user-input',
        schema_id: 'text-input-v1',
        data: {
          content,
          style: 'creative',
        },
        bucket_ids: [selectedChapter],
      });

      Alert.alert('Success', 'Content saved!');
      setContent('');
    } catch (error) {
      Alert.alert('Error', 'Failed to save content');
    }
  };

  const handleExpand = async () => {
    if (!content) return;

    setIsExpanding(true);
    try {
      // This triggers text expansion processor
      const blob = await createBlob({
        processor_id: 'user-input',
        schema_id: 'text-input-v1',
        data: {
          content,
          style: 'creative',
        },
        bucket_ids: [selectedChapter],
      });

      // Wait for expanded content via WebSocket
      // In real app, this would be handled by WebSocket listener
      setTimeout(() => {
        setExpandedContent(content + '\n\n[Expanded content would appear here...]');
        setIsExpanding(false);
      }, 2000);
    } catch (error) {
      setIsExpanding(false);
      Alert.alert('Error', 'Failed to expand content');
    }
  };

  const handleExport = async () => {
    try {
      const exported = await exportBucket(bookId, 'text');
      // Save to file or share
      Alert.alert('Success', 'Book exported successfully!');
    } catch (error) {
      Alert.alert('Error', 'Failed to export book');
    }
  };

  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>{book?.name || 'Loading...'}</Text>
        <TouchableOpacity onPress={handleExport} style={styles.exportButton}>
          <Text style={styles.exportButtonText}>Export</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.splitView}>
        <View style={styles.leftPanel}>
          <Text style={styles.sectionTitle}>Chapters</Text>
          <BucketTree
            bucket={book}
            onSelect={(chapterId) => setSelectedChapter(chapterId)}
            selectedId={selectedChapter}
          />
        </View>

        <View style={styles.rightPanel}>
          <Text style={styles.sectionTitle}>Content Editor</Text>
          
          <TextInput
            style={styles.editor}
            value={content}
            onChangeText={setContent}
            placeholder="Write your content here..."
            multiline
            textAlignVertical="top"
          />

          <View style={styles.buttonRow}>
            <TouchableOpacity 
              onPress={handleSaveContent}
              style={[styles.button, styles.saveButton]}
            >
              <Text style={styles.buttonText}>Save</Text>
            </TouchableOpacity>

            <TouchableOpacity 
              onPress={handleExpand}
              style={[styles.button, styles.expandButton]}
              disabled={isExpanding}
            >
              <Text style={styles.buttonText}>
                {isExpanding ? 'Expanding...' : 'Expand with AI'}
              </Text>
            </TouchableOpacity>
          </View>

          {expandedContent && (
            <View style={styles.expandedSection}>
              <Text style={styles.sectionTitle}>Expanded Content</Text>
              <Text style={styles.expandedText}>{expandedContent}</Text>
            </View>
          )}
        </View>
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: 16,
    backgroundColor: 'white',
    borderBottomWidth: 1,
    borderBottomColor: '#e0e0e0',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
  },
  exportButton: {
    backgroundColor: '#007AFF',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
  },
  exportButtonText: {
    color: 'white',
    fontWeight: '600',
  },
  splitView: {
    flexDirection: 'row',
    flex: 1,
  },
  leftPanel: {
    width: '30%',
    backgroundColor: 'white',
    padding: 16,
    borderRightWidth: 1,
    borderRightColor: '#e0e0e0',
  },
  rightPanel: {
    flex: 1,
    padding: 16,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    marginBottom: 12,
  },
  editor: {
    backgroundColor: 'white',
    borderRadius: 8,
    padding: 12,
    minHeight: 200,
    fontSize: 16,
    borderWidth: 1,
    borderColor: '#e0e0e0',
  },
  buttonRow: {
    flexDirection: 'row',
    marginTop: 16,
    gap: 12,
  },
  button: {
    flex: 1,
    paddingVertical: 12,
    borderRadius: 8,
    alignItems: 'center',
  },
  saveButton: {
    backgroundColor: '#4CAF50',
  },
  expandButton: {
    backgroundColor: '#9C27B0',
  },
  buttonText: {
    color: 'white',
    fontWeight: '600',
    fontSize: 16,
  },
  expandedSection: {
    marginTop: 24,
  },
  expandedText: {
    backgroundColor: 'white',
    padding: 12,
    borderRadius: 8,
    fontSize: 16,
    lineHeight: 24,
  },
});
```

## Task 10: Create Main App Component
**File**: `App.tsx`
```tsx
import React from 'react';
import { Provider } from 'react-redux';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { NavigationContainer } from '@react-navigation/native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { store } from './src/store';
import { AppNavigator } from './src/navigation/AppNavigator';

const queryClient = new QueryClient();

export default function App() {
  return (
    <Provider store={store}>
      <QueryClientProvider client={queryClient}>
        <SafeAreaProvider>
          <NavigationContainer>
            <AppNavigator />
          </NavigationContainer>
        </SafeAreaProvider>
      </QueryClientProvider>
    </Provider>
  );
}
```

## Task 11: Configure for Web
**File**: `webpack.config.js`
```javascript
const createExpoWebpackConfigAsync = require('@expo/webpack-config');

module.exports = async function (env, argv) {
  const config = await createExpoWebpackConfigAsync(
    {
      ...env,
      babel: {
        dangerouslyAddModulePathsToTranspile: [
          'react-native-vector-icons',
          'react-native-elements',
        ],
      },
    },
    argv
  );

  // Customize the config
  config.resolve.alias = {
    ...config.resolve.alias,
    'react-native$': 'react-native-web',
  };

  return config;
};
```

## Task 12: Create Responsive Styles
**File**: `src/theme/responsive.ts`
```typescript
import { Dimensions, Platform } from 'react-native';

const { width, height } = Dimensions.get('window');

export const isWeb = Platform.OS === 'web';
export const isMobile = width < 768;
export const isTablet = width >= 768 && width < 1024;
export const isDesktop = width >= 1024;

export const responsive = {
  width: (percentage: number) => (width * percentage) / 100,
  height: (percentage: number) => (height * percentage) / 100,
  
  fontSize: (size: number) => {
    if (isMobile) return size * 0.9;
    if (isTablet) return size;
    return size * 1.1;
  },
  
  padding: (size: number) => {
    if (isMobile) return size * 0.8;
    if (isTablet) return size;
    return size * 1.2;
  },
};

export const breakpoints = {
  mobile: 0,
  tablet: 768,
  desktop: 1024,
  wide: 1440,
};
```

## Task 13: Package Scripts
**File**: `package.json`
```json
{
  "scripts": {
    "start": "expo start",
    "android": "expo start --android",
    "ios": "expo start --ios",
    "web": "expo start --web",
    "build:web": "expo build:web",
    "build:android": "eas build --platform android",
    "build:ios": "eas build --platform ios",
    "test": "jest",
    "lint": "eslint . --ext .ts,.tsx",
    "type-check": "tsc --noEmit"
  }
}
```

## Testing Checklist
- [ ] App runs on web browser
- [ ] App runs on iOS simulator
- [ ] App runs on Android emulator
- [ ] Authentication flow works
- [ ] WebSocket connection establishes
- [ ] Book creation and editing works
- [ ] Text expansion triggers and displays
- [ ] Export functionality works
- [ ] Responsive design adapts to screen sizes
- [ ] Dynamic components render from schemas

## Success Criteria
- [ ] Single codebase runs on all platforms
- [ ] Real-time updates via WebSocket
- [ ] Dynamic UI generation from schemas
- [ ] Smooth performance on mobile
- [ ] Offline support with data sync
- [ ] Export books as text files