import React, { useEffect, useState, useRef, useCallback, useMemo } from "react";
import { Button, Card, CardBody, CardHeader, Divider, Tooltip, Spinner, Skeleton, Tabs, Tab, Dropdown, DropdownTrigger, DropdownMenu, DropdownItem } from "@nextui-org/react";
import { HeartIcon } from './HeartIcon';
import { CameraIcon } from './CameraIcon';
import { BsTranslate } from "react-icons/bs";
import { MdContentCopy, MdVolumeUp, MdPushPin, MdOutlinePushPin, MdLightbulb } from "react-icons/md";
import { ToolBarShow, Show, Hide, SetToolBarPinned, GetToolBarPinned, TranslateStream, Translate, TranslateMeanings, ExplainStream, GetExplainTemplates, SetDefaultExplainTemplate } from "../../../bindings/handy-translate/app";
import { lingva_tts } from "../../services/tts";
import { useVoice } from "../../hooks/useVoice";
import { Events, Window } from "@wailsio/runtime";
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';

// 常量配置
const CONSTANTS = {
    LOADING_HEIGHT: 50,
    MAX_CONTENT_HEIGHT: 500,
    DEBOUNCE_DELAY: 50,
    HIDE_DELAY: 100,
    COPY_RESET_DELAY: 2000,
    PLAYING_RESET_DELAY: 1000,
    WORD_MAX_LENGTH: 20,
    WORD_REGEX: /^[a-zA-Z]{1,20}$/,
    CHINESE_REGEX: /[\u4e00-\u9fa5]/,
    HIGHLIGHT_REGEX: (word) => new RegExp(`\\b(${word}[a-z]*)\\b`, 'gi'),
    BASE64_REGEX: /^[A-Za-z0-9+/]*={0,2}$/,
    PART_OF_SPEECH_STYLES: {
        'noun': 'bg-blue-100 text-blue-800',
        'verb': 'bg-green-100 text-green-800',
        'adjective': 'bg-purple-100 text-purple-800',
        'adverb': 'bg-orange-100 text-orange-800',
        'pronoun': 'bg-pink-100 text-pink-800',
        'preposition': 'bg-yellow-100 text-yellow-800',
        'conjunction': 'bg-indigo-100 text-indigo-800',
        'interjection': 'bg-red-100 text-red-800',
    },
    PART_OF_SPEECH_ABBRS: {
        'noun': 'n.',
        'verb': 'v.',
        'adjective': 'adj.',
        'adverb': 'adv.',
        'pronoun': 'pron.',
        'preposition': 'prep.',
        'conjunction': 'conj.',
        'interjection': 'interj.',
    }
};


export default function ToolBar() {
    const [result, setResult] = useState("")
    const [resultStream, setResultStream] = useState("")
    const [resultMeaningsStream, setResultMeaningsStream] = useState("")
    const [queryText, setQueryText] = useState("") // 原始查询文本
    const [isWord, setIsWord] = useState(false) // 是否为单词
    const [wordDetails, setWordDetails] = useState(null) // 词典详情
    const [translatedDefinitions, setTranslatedDefinitions] = useState({}) // 翻译后的释义 {key: translation}
    const [translatedExamples, setTranslatedExamples] = useState({}) // 翻译后的例句 {key: translation}
    const streamBufferRef = useRef(''); // 流式缓冲区
    const [isLoading, setIsLoading] = useState(false)
    const [isCopied, setIsCopied] = useState(false)
    const [isPlaying, setIsPlaying] = useState(false)
    const [isPlayingEn, setIsPlayingEn] = useState(false) // 播放英文
    const [isPlayingZh, setIsPlayingZh] = useState(false) // 播放中文
    const [isPinned, setIsPinned] = useState(false) // 是否固定窗口
    const [isAnimating, setIsAnimating] = useState(true) // 动画状态
    const [mode, setMode] = useState('translate') // 模式：translate/explain
    const modeRef = useRef('translate') // 用于在事件处理函数中访问最新的 mode 值
    const [explainTemplates, setExplainTemplates] = useState([]) // 解释模板列表
    const [selectedTemplate, setSelectedTemplate] = useState('') // 选中的模板ID
    const selectedTemplateRef = useRef('') // 用于在事件处理函数中访问最新的 selectedTemplate 值
    const [defaultTemplate, setDefaultTemplate] = useState('') // 默认模板ID
    const playOrStop = useVoice()
    const contentRef = useRef(); // 实际内容容器的引用
    const { t } = useTranslation(); // 国际化

    // 初始化时从后端获取固定状态和模板列表
    useEffect(() => {
        GetToolBarPinned().then(pinned => {
            console.log('从后端获取工具栏固定状态:', pinned)
            setIsPinned(pinned)
            // 如果已固定，设置窗口为始终置顶
            if (pinned) {
                Window.SetAlwaysOnTop(true)
            }
        }).catch(err => {
            console.error('获取固定状态失败:', err)
        })

        // 获取解释模板列表
        GetExplainTemplates().then(result => {
            try {
                const data = JSON.parse(result)
                console.log('获取到解释模板:', data)

                if (data.templates && Object.keys(data.templates).length > 0) {
                    // 转换为数组格式
                    const templatesArray = Object.keys(data.templates).map(id => ({
                        id,
                        ...data.templates[id]
                    }))
                    setExplainTemplates(templatesArray)

                    // 设置默认模板
                    const defaultId = data.default_template || templatesArray[0]?.id || ''
                    setDefaultTemplate(defaultId)
                    setSelectedTemplate(defaultId)
                    selectedTemplateRef.current = defaultId // 同步更新 ref
                }
            } catch (err) {
                console.error('解析模板数据失败:', err)
            }
        }).catch(err => {
            console.error('获取解释模板失败:', err)
        })
    }, [])

    // 检测是否为单个单词
    const checkIsWord = useCallback((text) => {
        if (!text) return false
        // 确保 text 是字符串类型
        const str = typeof text === 'string' ? text : String(text)
        const trimmed = str.trim()
        // 单个单词：只包含字母，长度1-20，无空格
        return CONSTANTS.WORD_REGEX.test(trimmed)
    }, [])

    // 获取单词详细信息（使用 Free Dictionary API）
    const fetchWordDetails = async (word) => {
        try {
            const response = await fetch(`https://api.dictionaryapi.dev/api/v2/entries/en/${word}`)
            if (response.ok) {
                const data = await response.json()
                if (data && data.length > 0) {
                    return data[0] // 返回第一个结果
                }
            }
        } catch (err) {
            console.error('获取词典信息失败:', err)
        }
        return null
    }

    // 复制到剪贴板
    const handleCopy = async () => {
        if (!(result || resultStream || resultMeaningsStream)) return

        try {
            await navigator.clipboard.writeText((result || '') + (resultStream || ''))
            setIsCopied(true)
            setTimeout(() => setIsCopied(false), CONSTANTS.COPY_RESET_DELAY)
        } catch (err) {
            console.error('复制失败:', err)
        }
    }

    // 固定/取消固定窗口
    const handlePinToggle = async () => {
        const newPinnedState = !isPinned

        // 调用后端设置固定状态
        try {
            // 先设置窗口置顶状态
            if (newPinnedState) {
                Window.SetAlwaysOnTop(true)
            } else {
                Window.SetAlwaysOnTop(false)
            }

            // 保存到后端
            await SetToolBarPinned(newPinnedState)

            // 更新前端状态
            setIsPinned(newPinnedState)

            console.log('窗口固定状态已更新:', newPinnedState)
        } catch (err) {
            console.error('设置固定状态失败:', err)
            // 失败时恢复之前的状态
            Window.SetAlwaysOnTop(isPinned)
        }
    }

    // 安全的 Base64 解码
    const safeAtob = (base64String) => {
        try {
            console.log('原始数据类型:', typeof base64String);
            console.log('原始数据长度:', base64String?.length);
            console.log('是否为数组:', Array.isArray(base64String));
            console.log('是否为 Uint8Array:', base64String instanceof Uint8Array);
            console.log('constructor:', base64String?.constructor?.name);

            if (!base64String) {
                throw new Error('Base64 字符串为空');
            }

            // 如果已经是 Uint8Array 或类似的字节数组，直接返回
            if (base64String instanceof Uint8Array) {
                console.log('数据已经是 Uint8Array，直接返回');
                return base64String;
            }

            // 如果是类数组对象（有 length 但不是字符串），转换为 Uint8Array
            if (typeof base64String === 'object' && base64String.length !== undefined && typeof base64String !== 'string') {
                console.log('数据是类数组对象，转换为 Uint8Array');
                return new Uint8Array(base64String);
            }

            // 如果不是字符串，尝试转换
            let strData = typeof base64String === 'string' ? base64String : String(base64String);
            console.log('转换为字符串后的类型:', typeof strData);
            console.log('字符串前100个字符:', strData.substring(0, 100));

            // 清理 Base64 字符串
            let cleaned = strData.trim();

            // 移除可能的换行符和空格
            cleaned = cleaned.replace(/[\r\n\s]/g, '');

            // 如果包含 data URI 前缀，移除它
            if (cleaned.includes('base64,')) {
                const parts = cleaned.split('base64,');
                cleaned = parts[parts.length - 1];
                console.log('移除 data URI 前缀后长度:', cleaned.length);
            }

            // 检查是否是有效的 Base64 字符
            const base64Regex = /^[A-Za-z0-9+/]*={0,2}$/;
            if (!base64Regex.test(cleaned)) {
                console.error('不是有效的 Base64 字符串，包含非法字符');
                console.error('前200个字符:', cleaned.substring(0, 200));
                // 尝试移除非 Base64 字符
                cleaned = cleaned.replace(/[^A-Za-z0-9+/=]/g, '');
                console.log('清理非法字符后长度:', cleaned.length);
            }

            // 确保长度是4的倍数（Base64 要求）
            const remainder = cleaned.length % 4;
            if (remainder !== 0) {
                const padding = 4 - remainder;
                cleaned += '='.repeat(padding);
                console.log('添加填充字符数:', padding);
            }

            console.log('最终 Base64 长度:', cleaned.length);
            console.log('最终 Base64 前30个字符:', cleaned.substring(0, 30));
            console.log('最终 Base64 后30个字符:', cleaned.substring(cleaned.length - 30));

            const binaryString = atob(cleaned);
            console.log('atob 解码成功，二进制字符串长度:', binaryString.length);

            const bytes = new Uint8Array(binaryString.length);
            for (let i = 0; i < binaryString.length; i++) {
                bytes[i] = binaryString.charCodeAt(i);
            }

            console.log('字节数组创建成功，长度:', bytes.length);
            console.log('前10个字节:', Array.from(bytes.slice(0, 10)));

            return bytes;
        } catch (err) {
            console.error('Base64 解码失败详细信息:');
            console.error('错误名称:', err.name);
            console.error('错误消息:', err.message);
            console.error('错误堆栈:', err.stack);
            console.error('输入字符串类型:', typeof base64String);
            console.error('输入字符串长度:', base64String?.length);
            if (base64String && base64String.length > 0) {
                console.error('前200个字符:', base64String.substring(0, 200));
                console.error('后200个字符:', base64String.substring(Math.max(0, base64String.length - 200)));
            }
            throw new Error(`Base64 解码失败: ${err.message}`);
        }
    }

    // 播放英文单词发音
    const handleSpeakEnglish = async () => {
        if (!queryText) {
            console.log('queryText 为空，无法播放')
            return
        }

        try {
            setIsPlayingEn(true)
            console.log('==== 开始播放英文单词 ====')
            console.log('单词:', queryText)
            console.log('语言: en')

            const audioData = await lingva_tts.tts(queryText, 'en')
            console.log('TTS API 返回:', audioData ? `成功 (长度: ${audioData.length})` : '失败/空')

            if (audioData) {
                console.log('开始 Base64 解码...')
                const bytes = safeAtob(audioData)
                console.log('解码完成，字节数:', bytes.length)
                console.log('开始播放音频...')
                await playOrStop(bytes)
                console.log('播放完成')
            } else {
                console.error('未获取到音频数据')
                alert('未获取到音频数据，请检查网络连接')
            }
        } catch (err) {
            console.error('英文播放失败:', err)
            alert(`播放失败: ${err.message}`)
        } finally {
            setTimeout(() => setIsPlayingEn(false), CONSTANTS.PLAYING_RESET_DELAY)
        }
    }

    // 播放中文翻译发音
    const handleSpeakChinese = async () => {
        if (!result) {
            console.log('result 为空，无法播放')
            return
        }

        try {
            setIsPlayingZh(true)
            console.log('==== 开始播放中文翻译 ====')
            console.log('文本:', result)
            console.log('语言: zh')

            const audioData = await lingva_tts.tts(result, 'zh')
            console.log('TTS API 返回:', audioData ? `成功 (长度: ${audioData.length})` : '失败/空')

            if (audioData) {
                console.log('开始 Base64 解码...')
                const bytes = safeAtob(audioData)
                console.log('解码完成，字节数:', bytes.length)
                console.log('开始播放音频...')
                await playOrStop(bytes)
                console.log('播放完成')
            } else {
                console.error('未获取到音频数据')
                alert('未获取到音频数据，请检查网络连接')
            }
        } catch (err) {
            console.error('中文播放失败:', err)
            alert(`播放失败: ${err.message}`)
        } finally {
            setTimeout(() => setIsPlayingZh(false), CONSTANTS.PLAYING_RESET_DELAY)
        }
    }

    // 语音播放（普通模式）
    const handleSpeak = async () => {
        if (!result) return

        try {
            setIsPlaying(true)
            const textToSpeak = result
            const lang = /[\u4e00-\u9fa5]/.test(result) ? 'zh' : 'en'
            console.log('普通模式播放，语言:', lang, '文本:', textToSpeak)

            const audioData = await lingva_tts.tts(textToSpeak, lang)

            if (audioData) {
                const bytes = safeAtob(audioData)
                await playOrStop(bytes)
            }
        } catch (err) {
            console.error('语音播放失败:', err)
            alert(`播放失败: ${err.message}`)
        } finally {
            setTimeout(() => setIsPlaying(false), CONSTANTS.PLAYING_RESET_DELAY)
        }
    }

    useEffect(() => {
        // 监听普通翻译结果
        const unsubscribeResult = Events.On("result", function (data) {
            // 确保 result 是字符串类型
            let result = typeof data.data === 'string' ? data.data : String(data.data || '')
            console.log('收到 result 事件:', { result, type: typeof data.data })
            setResult(result)
            // 非流式结果到达时，清理旧的流式内容，避免累积显示
            streamBufferRef.current = ''
            setResultStream('')
            setResultMeaningsStream("")
            // ✅ 清除加载状态
            setIsLoading(false)
            // 不在这里计算高度，统一在下面的 useEffect 中处理
        })

        // 监听 query 事件（流式翻译开始时重置）
        const unsubscribeQuery = Events.On("query", async function (data) {
            // 确保 text 是字符串类型
            const text = typeof data.data === 'string' ? data.data : String(data.data || '')
            console.log('收到 query 事件:', { text, type: typeof data.data, data: data.data })

            // ✅ 立即设置加载状态，防止窗口被隐藏
            setIsLoading(true)

            setQueryText(text)
            streamBufferRef.current = '' // 重置流式缓冲区
            setResult('') // 清空显示
            setResultStream('') // 清空流式缓冲区
            setResultMeaningsStream("")
            setWordDetails(null) // 清空词典信息

            // 检测是否为单词
            const isWordCheck = checkIsWord(text)
            setIsWord(isWordCheck)

            // 如果是单词，获取词典信息
            if (isWordCheck && modeRef.current === 'translate') {
                const details = await fetchWordDetails(text.trim())
                setWordDetails(details)
                // 重置翻译缓存
                setTranslatedDefinitions({})
                setTranslatedExamples({})

                // 异步翻译所有释义和例句
                if (details?.meanings) {
                    translateWordDetails(details)
                }
            }

            // 如果是解释模式，前端主动调用 ExplainStream（使用当前选中的模板）
            // 使用 ref 获取最新的值，避免闭包问题
            if (modeRef.current === 'explain') {
                const templateId = selectedTemplateRef.current || defaultTemplate || ''
                console.log('解释模式，主动调用 ExplainStream，templateID:', templateId, '当前 mode:', modeRef.current)
                await ExplainStream(text, templateId)
            }
            // 翻译模式由后端自动处理
        })

        // 监听流式翻译结果
        const unsubscribeStream = Events.On("result_stream", function (data) {
            // 确保 chunk 是字符串类型
            let chunk = typeof data.data === 'string' ? data.data : String(data.data || '')
            console.log('ToolBar 收到流式数据块:', chunk, '当前总长度:', streamBufferRef.current.length)

            // ✅ 收到第一个数据块时，清除加载状态
            if (streamBufferRef.current.length === 0 && chunk) {
                setIsLoading(false)
            }

            streamBufferRef.current += chunk // 累积到 ref
            setResultStream(streamBufferRef.current) // 更新状态触发重渲染
        })
        const unsubscribeMeaningsStream = Events.On("result_meanings_stream", function (data) {
            // 确保 chunk 是字符串类型
            let chunk = typeof data.data === 'string' ? data.data : String(data.data || '')
            console.log('ToolBar 收到流式数据块:', chunk, '当前总长度:', streamBufferRef.current.length)

            // ✅ 收到第一个数据块时，清除加载状态
            if (streamBufferRef.current.length === 0 && chunk) {
                setIsLoading(false)
            }

            streamBufferRef.current += chunk // 累积到 ref
            setResultMeaningsStream(streamBufferRef.current) // 更新状态触发重渲染
        })

        // 监听流式完成
        const unsubscribeStreamDone = Events.On("result_stream_done", function (data) {
            console.log('ToolBar 流式翻译完成')
            // ✅ 确保清除加载状态
            setIsLoading(false)
        })

        // 清理事件监听
        return () => {
            if (unsubscribeResult) unsubscribeResult()
            if (unsubscribeQuery) unsubscribeQuery()
            if (unsubscribeStream) unsubscribeStream()
            if (unsubscribeStreamDone) unsubscribeStreamDone()
            if (unsubscribeMeaningsStream) unsubscribeMeaningsStream()
        }
    }, [])

    useEffect(() => {
        // 检查是否有内容或正在加载
        const hasContent = !!(result || resultStream || resultMeaningsStream || wordDetails || isLoading)

        if (!hasContent) {
            // 无内容且未加载时隐藏窗口
            const timer = setTimeout(() => {
                console.log('无内容，隐藏工具栏窗口')
                Hide("ToolBar").catch(err => {
                    console.log('隐藏窗口失败（可能窗口尚未初始化）:', err)
                })
            }, 100)
            return () => clearTimeout(timer)
        }

        // 如果正在加载，显示固定高度的加载窗口
        if (isLoading) {
            const loadingHeight = 50 // 加载动画固定高度
            console.log('显示加载动画，高度:', loadingHeight)
            ToolBarShow(loadingHeight)
            return
        }

        // 使用防抖延迟来避免流式翻译时频繁更新
        const debounceTimer = setTimeout(() => {
            // 使用双重 requestAnimationFrame 确保 DOM 完全更新后再计算高度
            // 第一个 RAF 等待 React 渲染完成
            // 第二个 RAF 等待浏览器布局计算完成
            requestAnimationFrame(() => {
                requestAnimationFrame(() => {
                    if (!contentRef.current) {
                        console.log('contentRef 未就绪')
                        return
                    }

                    // 获取实际渲染内容的高度
                    const contentHeight = contentRef.current.scrollHeight

                    // CardHeader 高度约 52px，Divider 1px，CardBody 的实际内容高度
                    const maxContentHeight = 500 // 最大内容高度
                    const actualContentHeight = Math.min(contentHeight, maxContentHeight)

                    console.log('有内容，显示工具栏:', {
                        contentHeight,
                        actualContentHeight,
                        isWord,
                        hasResult: !!result,
                        resultLength: result?.length || 0
                    })

                    // 调用 ToolBarShow 会自动显示窗口并设置高度
                    ToolBarShow(actualContentHeight)
                })
            })
        }, 50) // 50ms 防抖延迟

        return () => clearTimeout(debounceTimer)
    }, [result, resultStream, resultMeaningsStream, isWord, wordDetails, isLoading]);

    // 获取词性标签样式
    const getPartOfSpeechStyle = (partOfSpeech) => {
        const styles = {
            'noun': 'bg-blue-100 text-blue-800',
            'verb': 'bg-green-100 text-green-800',
            'adjective': 'bg-purple-100 text-purple-800',
            'adverb': 'bg-orange-100 text-orange-800',
            'pronoun': 'bg-pink-100 text-pink-800',
            'preposition': 'bg-yellow-100 text-yellow-800',
            'conjunction': 'bg-indigo-100 text-indigo-800',
            'interjection': 'bg-red-100 text-red-800',
        }
        return styles[partOfSpeech] || 'bg-gray-100 text-gray-800'
    }

    // 获取词性缩写
    const getPartOfSpeechAbbr = (partOfSpeech) => {
        const abbrs = {
            'noun': 'n.',
            'verb': 'v.',
            'adjective': 'adj.',
            'adverb': 'adv.',
            'pronoun': 'pron.',
            'preposition': 'prep.',
            'conjunction': 'conj.',
            'interjection': 'interj.',
        }
        return abbrs[partOfSpeech] || partOfSpeech + '.'
    }

    // 翻译单词详情（释义和例句）- 并行翻译以提高速度
    const translateWordDetails = async (details) => {
        if (!details?.meanings) return

        // 收集所有需要翻译的文本
        const translationTasks = []

        details.meanings.forEach((meaning, meaningIdx) => {
            if (meaning.definitions) {
                meaning.definitions.forEach((def, defIdx) => {
                    const key = `${meaningIdx}-${defIdx}`

                    // 翻译释义
                    if (def.definition) {
                        translationTasks.push({
                            key,
                            text: def.definition,
                            type: 'definition'
                        })
                    }

                    // 翻译例句
                    if (def.example) {
                        translationTasks.push({
                            key,
                            text: def.example,
                            type: 'example'
                        })
                    }
                })
            }
        })

        // 并行翻译所有文本
        const translationPromises = translationTasks.map(async (task) => {
            try {
                const translation = await TranslateMeanings(task.text, 'en', 'zh')
                if (task.type === 'definition') {
                    setTranslatedDefinitions(prev => ({
                        ...prev,
                        [task.key]: translation
                    }))
                } else {
                    setTranslatedExamples(prev => ({
                        ...prev,
                        [task.key]: translation
                    }))
                }
            } catch (err) {
                console.error(`翻译${task.type === 'definition' ? '释义' : '例句'}失败:`, err)
            }
        })

        // 等待所有翻译完成（不阻塞UI）
        Promise.all(translationPromises).catch(err => {
            console.error('批量翻译出错:', err)
        })
    }

    // 播放例句发音
    const handleSpeakExample = async (exampleText) => {
        if (!exampleText) return
        try {
            const audioData = await lingva_tts.tts(exampleText, 'en')
            if (audioData) {
                const bytes = safeAtob(audioData)
                await playOrStop(bytes)
            }
        } catch (err) {
            console.error('播放例句失败:', err)
        }
    }

    // 将单词在例句中高亮
    const highlightWord = (text, word) => {
        if (!text || !word) return text
        const regex = new RegExp(`\\b(${word}[a-z]*)\\b`, 'gi')
        const parts = []
        let lastIndex = 0
        let match

        const regexGlobal = new RegExp(`\\b(${word}[a-z]*)\\b`, 'gi')
        while ((match = regexGlobal.exec(text)) !== null) {
            // 添加匹配前的文本
            if (match.index > lastIndex) {
                parts.push(text.substring(lastIndex, match.index))
            }
            // 添加高亮的匹配文本
            parts.push(<span key={match.index} className="text-red-500 font-medium">{match[0]}</span>)
            lastIndex = match.index + match[0].length
        }
        // 添加剩余文本
        if (lastIndex < text.length) {
            parts.push(text.substring(lastIndex))
        }
        return parts.length > 0 ? parts : text
    }

    // 渲染加载动画
    const renderLoading = () => {
        return (
            <div className="loading-container p-6 flex flex-col items-center justify-center space-y-4 min-h-[100px]">
                {/* <div className="flex items-center gap-3">
                    <Spinner
                        size="lg"
                        color="primary"
                    />
                    <span className="text-primary text-sm font-medium">正在翻译...</span>
                </div> */}
                <div className="space-y-3 w-full px-2">
                    <Skeleton className="rounded-lg loading-skeleton">
                        <div className="h-3 w-4/5 rounded-lg bg-default-200"></div>
                    </Skeleton>
                    <Skeleton className="rounded-lg loading-skeleton" style={{ animationDelay: '0.2s' }}>
                        <div className="h-3 w-full rounded-lg bg-default-200"></div>
                    </Skeleton>
                    <Skeleton className="rounded-lg loading-skeleton" style={{ animationDelay: '0.4s' }}>
                        <div className="h-3 w-3/5 rounded-lg bg-default-300"></div>
                    </Skeleton>
                </div>
            </div>
        )
    }

    // 渲染词典格式内容
    const renderWordDetailsContent = () => {
        console.log('renderWordDetailsContent 调用，wordDetails:', wordDetails, 'queryText:', queryText, 'result:', result, "resultStream:", resultStream)

        return (
            <>
                {/* 单词 + 音标 + 播放按钮 */}
                <div className="flex items-center gap-2 mb-3">
                    <h3 className="text-xl font-bold text-black">{queryText}</h3>
                    {wordDetails?.phonetic && (
                        <>
                            <span className="text-sm text-gray-600">/{wordDetails.phonetic.replace(/[\/\[\]]/g, '')}/</span>
                        </>
                    )}
                    {/* 播放按钮始终显示 */}
                    <Button
                        size="sm"
                        isIconOnly
                        variant="light"
                        aria-label="Play English"
                        onPress={handleSpeakEnglish}
                        isLoading={isPlayingEn}
                        className="shrink-0"
                    >
                        <MdVolumeUp className="text-base text-gray-600" />
                    </Button>
                </div>

                {/* 词性和释义 */}
                {wordDetails?.meanings && wordDetails.meanings.length > 0 ? (
                    wordDetails.meanings.map((meaning, idx) => (
                        <div key={idx} className={`mb-4 ${idx > 0 ? 'pt-3 border-t border-gray-200' : ''}`}>
                            {/* 词性标签 - 使用彩色标签样式 */}
                            <div className="mb-2">
                                <span className={`inline-block px-2.5 py-1 rounded-md text-xs font-semibold ${getPartOfSpeechStyle(meaning.partOfSpeech)}`}>
                                    {getPartOfSpeechAbbr(meaning.partOfSpeech)}
                                </span>
                            </div>

                            {/* 释义列表 */}
                            <div className="ml-0 space-y-2">
                                {meaning.definitions.map((def, defIdx) => {
                                    const translationKey = `${idx}-${defIdx}`
                                    const definitionTranslation = translatedDefinitions[translationKey]
                                    const exampleTranslation = translatedExamples[translationKey]

                                    return (
                                        <div key={defIdx} className="mb-3 last:mb-0">
                                            {/* 英文释义 + 中文翻译 */}
                                            <div className="mb-1.5">
                                                <p className="text-sm text-black leading-relaxed mb-1">{def.definition}</p>
                                                {definitionTranslation ? (
                                                    <p className="text-sm text-gray-600 leading-relaxed ml-1">{definitionTranslation}</p>
                                                ) : (
                                                    <div className="ml-1 h-4 w-20 bg-gray-200 animate-pulse rounded"></div>
                                                )}
                                            </div>

                                            {/* 英文例句 + 中文翻译 - 带播放按钮 */}
                                            {def.example && (
                                                <div className="ml-1 mt-2 p-2.5 bg-gray-50 rounded-md border-l-4 border-blue-400 hover:bg-gray-100 transition-colors">
                                                    <div className="space-y-1.5">
                                                        <div className="flex items-start gap-2">
                                                            <div className="flex-1">
                                                                <p className="text-sm text-gray-700 leading-relaxed italic">
                                                                    "{highlightWord(def.example, queryText)}"
                                                                </p>
                                                            </div>
                                                            <Tooltip content="播放例句" placement="top">
                                                                <Button
                                                                    size="sm"
                                                                    isIconOnly
                                                                    variant="light"
                                                                    aria-label="Play Example"
                                                                    onPress={() => handleSpeakExample(def.example)}
                                                                    className="shrink-0 h-6 w-6 min-w-6 hover:bg-gray-200"
                                                                >
                                                                    <MdVolumeUp className="text-xs text-gray-600" />
                                                                </Button>
                                                            </Tooltip>
                                                        </div>
                                                        {/* 中文翻译 */}
                                                        {exampleTranslation ? (
                                                            <p className="text-sm text-gray-600 leading-relaxed ml-1">
                                                                "{exampleTranslation}"
                                                            </p>
                                                        ) : (
                                                            <div className="ml-1 h-4 w-32 bg-gray-200 animate-pulse rounded"></div>
                                                        )}
                                                    </div>
                                                </div>
                                            )}

                                            {def.word && (
                                                <div className="pt-2 mt-2 border-t border-gray-200 flex items-center justify-between gap-2">
                                                    <p className="text-sm text-black font-medium flex-1 leading-relaxed">{exampleTranslation}</p>
                                                    <Button
                                                        size="sm"
                                                        isIconOnly
                                                        variant="light"
                                                        aria-label="Play Chinese"
                                                        onPress={handleSpeakChinese}
                                                        isLoading={isPlayingZh}
                                                        className="shrink-0"
                                                    >
                                                        <MdVolumeUp className="text-base text-gray-600" />
                                                    </Button>
                                                </div>
                                            )}
                                        </div>
                                    )
                                })}
                            </div>
                        </div>
                    ))
                ) : (
                    // 如果没有词典数据，显示提示
                    <div className="mb-2 text-sm text-gray-500">
                        词典暂无详细释义
                    </div>
                )}

                {/* 底部中文翻译 - 始终显示 */}
                {resultStream && (
                    <div className="pt-2 mt-2 border-t border-gray-200 flex items-center justify-between gap-2">
                        <p className="text-sm text-black font-medium flex-1 leading-relaxed">{resultStream}</p>
                        <Button
                            size="sm"
                            isIconOnly
                            variant="light"
                            aria-label="Play Chinese"
                            onPress={handleSpeakChinese}
                            isLoading={isPlayingZh}
                            className="shrink-0"
                        >
                            <MdVolumeUp className="text-base text-gray-600" />
                        </Button>
                    </div>
                )}
            </>
        )
    }

    return (
        <Card
            shadow="lg"
            className='rounded-xl border-none shadow-xl w-full'>

            <CardHeader style={{ "--wails-draggable": "drag" }}>

                <div className="flex gap-2 items-center w-full justify-between" style={{ WebkitAppRegion: 'drag' }}>
                    <Tooltip content="打开翻译窗口" placement="bottom">
                        <Button size="sm" isIconOnly color="danger" aria-label="Translate" onPress={() => {
                            Show("Translate")
                        }}>
                            <BsTranslate />
                        </Button>
                    </Tooltip>

                    {/* 翻译/解释模式切换 */}
                    <Tabs
                        selectedKey={mode}
                        onSelectionChange={async (key) => {
                            setMode(key)
                            modeRef.current = key // 同步更新 ref
                            // 通知后端更新模式
                            Events.Emit({ name: "toolbarMode", data: key })

                            // 切换模式后，如果有queryText，重新调用对应API
                            if (queryText && queryText.trim() !== '') {
                                setIsLoading(true)
                                setResult('')
                                setResultStream('')
                                setResultMeaningsStream('')
                                streamBufferRef.current = ''

                                if (key === 'translate') {
                                    console.log('切换到翻译模式，调用 TranslateStream')
                                    // 默认使用 auto 和 zh
                                    await TranslateStream(queryText, 'auto', 'zh')
                                } else if (key === 'explain') {
                                    console.log('切换到解释模式，调用 ExplainStream')
                                    console.log('selectedTemplate:', selectedTemplate, 'defaultTemplate:', defaultTemplate)
                                    setWordDetails(null)
                                    // 使用选中的模板ID，如果没有则使用默认模板
                                    const templateId = selectedTemplate || defaultTemplate || ''
                                    console.log('使用的 templateID:', templateId)
                                    await ExplainStream(queryText, templateId)
                                }
                            }
                        }}
                        size="sm"
                        aria-label="Mode Switch"
                    >
                        <Tab
                            key="translate"
                            title={
                                <div className="flex items-center gap-1">
                                    <BsTranslate className="text-xs" />
                                    <span className="text-xs">{t('translate.translate')}</span>
                                </div>
                            }
                        />
                        <Tab
                            key="explain"
                            title={
                                <div className="flex items-center gap-1">
                                    <MdLightbulb className="text-xs" />
                                    <span className="text-xs">{t('translate.explain')}</span>
                                </div>
                            }
                        />
                    </Tabs>

                    {/* 解释模式下的模板选择器 */}
                    {mode === 'explain' && explainTemplates.length > 0 && (
                        <Dropdown placement="top">
                            <DropdownTrigger>
                                <Button
                                    size="sm"
                                    variant="flat"
                                    className="min-w-[100px]"
                                >
                                    {explainTemplates.find(t => t.id === selectedTemplate)?.name || explainTemplates.find(t => t.id === defaultTemplate)?.name || t('translate.template_placeholder')}
                                </Button>
                            </DropdownTrigger>
                            <DropdownMenu
                                aria-label={t('translate.select_template')}
                                selectedKeys={selectedTemplate ? [selectedTemplate] : []}
                                selectionMode="single"
                                onAction={async (key) => {
                                    const newTemplateId = String(key)
                                    console.log('模板切换，新模板ID:', newTemplateId)
                                    setSelectedTemplate(newTemplateId)
                                    selectedTemplateRef.current = newTemplateId // 同步更新 ref

                                    // 切换模板后，如果有查询文本，重新解释
                                    if (queryText && queryText.trim() !== '') {
                                        setIsLoading(true)
                                        setResult('')
                                        setResultStream('')
                                        setResultMeaningsStream('')
                                        streamBufferRef.current = ''
                                        console.log('调用 ExplainStream，queryText:', queryText, 'templateID:', newTemplateId)
                                        await ExplainStream(queryText, newTemplateId)
                                    }
                                }}
                                className="max-h-[40vh] overflow-y-auto"
                            >
                                {explainTemplates.map((template) => (
                                    <DropdownItem
                                        key={template.id}
                                        description={template.description}
                                    >
                                        {template.name}
                                    </DropdownItem>
                                ))}
                            </DropdownMenu>
                        </Dropdown>
                    )}

                    <div className="flex gap-2">
                        <Tooltip content={isPinned ? "取消固定" : "固定窗口"} placement="bottom">
                            <Button
                                size="sm"
                                isIconOnly
                                color={isPinned ? "warning" : "default"}
                                variant={isPinned ? "solid" : "flat"}
                                aria-label="Pin"
                                onPress={handlePinToggle}
                            >

                                {isPinned ? <MdPushPin /> : <MdOutlinePushPin />}
                            </Button>
                        </Tooltip>

                        <Tooltip content={isCopied ? "已复制!" : "复制"} placement="bottom">
                            <Button
                                size="sm"
                                isIconOnly
                                color={isCopied ? "success" : "primary"}
                                aria-label="Copy"
                                onPress={handleCopy}
                                isDisabled={!(result || resultStream)}
                            >
                                <MdContentCopy />
                            </Button>
                        </Tooltip>

                        <Tooltip content={isPlaying ? "播放中..." : "朗读"} placement="bottom">
                            <Button
                                size="sm"
                                isIconOnly
                                color="secondary"
                                aria-label="Speak"
                                onPress={handleSpeak}
                                isDisabled={!result}
                                isLoading={isPlaying}
                            >
                                <MdVolumeUp />
                            </Button>
                        </Tooltip>
                    </div>
                </div>
            </CardHeader>
            <Divider />

            <CardBody className="overflow-hidden">
                {isLoading ? (
                    // 加载动画
                    renderLoading()
                ) : (
                    // 翻译内容
                    <div ref={contentRef} className={`${isWord ? '' : 'p-4'} max-h-[500px] overflow-y-auto`}>
                        {isWord && mode !== 'explain' ? (
                            // 词典格式显示（即使没有详细释义也显示）
                            <div className="p-4">
                                {renderWordDetailsContent()}
                            </div>
                        ) : mode === 'explain' ? (
                            // 解释模式：使用 Markdown 渲染
                            <div className="markdown-content p-4 text-black leading-relaxed">
                                <ReactMarkdown
                                    components={{
                                        // 自定义样式组件
                                        h1: ({node, ...props}) => <h1 className="text-2xl font-bold mb-3 mt-4" {...props} />,
                                        h2: ({node, ...props}) => <h2 className="text-xl font-bold mb-2 mt-3" {...props} />,
                                        h3: ({node, ...props}) => <h3 className="text-lg font-bold mb-2 mt-3" {...props} />,
                                        h4: ({node, ...props}) => <h4 className="text-base font-bold mb-1 mt-2" {...props} />,
                                        p: ({node, ...props}) => <p className="mb-3 leading-relaxed" {...props} />,
                                        ul: ({node, ...props}) => <ul className="list-disc list-inside mb-3 space-y-1 ml-4" {...props} />,
                                        ol: ({node, ...props}) => <ol className="list-decimal list-inside mb-3 space-y-1 ml-4" {...props} />,
                                        li: ({node, ...props}) => <li className="leading-relaxed" {...props} />,
                                        code: ({node, inline, ...props}) => 
                                            inline ? (
                                                <code className="bg-gray-100 text-red-600 px-1.5 py-0.5 rounded text-sm font-mono" {...props} />
                                            ) : (
                                                <code className="block bg-gray-100 text-gray-800 p-3 rounded mb-3 overflow-x-auto font-mono text-sm whitespace-pre" {...props} />
                                            ),
                                        pre: ({node, ...props}) => <pre className="bg-gray-100 p-3 rounded mb-3 overflow-x-auto" {...props} />,
                                        blockquote: ({node, ...props}) => <blockquote className="border-l-4 border-gray-300 pl-4 italic my-3 text-gray-700" {...props} />,
                                        strong: ({node, ...props}) => <strong className="font-bold" {...props} />,
                                        em: ({node, ...props}) => <em className="italic" {...props} />,
                                        hr: ({node, ...props}) => <hr className="my-4 border-gray-300" {...props} />,
                                    }}
                                >
                                    {resultStream || result}
                                </ReactMarkdown>
                            </div>
                        ) : (
                            // 普通翻译结果
                            <p className="text-black leading-relaxed whitespace-pre-wrap p-4">
                                {resultStream || result}
                            </p>
                        )}
                    </div>
                )}
            </CardBody>

        </Card >
    );
}
