import React, { useEffect, useState, useRef, useCallback, useMemo } from "react";
import { Button, Card, CardBody, CardHeader, Divider, Tooltip, Spinner, Skeleton, Tabs, Tab, Dropdown, DropdownTrigger, DropdownMenu, DropdownItem, Popover, PopoverTrigger, PopoverContent } from "@nextui-org/react";
import { HeartIcon } from './HeartIcon';
import { CameraIcon } from './CameraIcon';
import { BsTranslate } from "react-icons/bs";
import { MdContentCopy, MdVolumeUp, MdPushPin, MdOutlinePushPin, MdLightbulb, MdCheck, MdPalette, MdKeyboardArrowDown, MdSearch } from "react-icons/md";
import { ToolBarShow, Show, Hide, SetToolBarPinned, GetToolBarPinned, Translate, TranslateMeanings, GetToolbarMode, GetExplainTemplates, SetDefaultExplainTemplate } from "../../../bindings/handy-translate/internal/app/binding";
import { lingva_tts } from "../../services/tts";
import { useVoice } from "../../hooks/useVoice";
import { Events, Window } from "@wailsio/runtime";
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import { themes, themeIds, applyThemeCssVars, getSavedThemeId, saveThemeId } from '../../themes';

// 常量配置
const CONSTANTS = {
    LOADING_HEIGHT: 50,
    MAX_CONTENT_HEIGHT: 500,
    DEBOUNCE_DELAY: 50,
    HIDE_DELAY: 100,
    COPY_RESET_DELAY: 2000,
    PLAYING_RESET_DELAY: 200,
    WORD_REGEX: /^[a-zA-Z](?:[a-zA-Z'-]{0,28}[a-zA-Z])?$/,
}

// TTS 内存缓存：相同文本第二次播放直接返回，无需网络请求
const ttsCache = new Map()
const TTS_CACHE_LIMIT = 50

function parseWordDetails(rawResult) {
    const result = typeof rawResult === 'string' ? rawResult.trim() : String(rawResult || '').trim()
    if (!result) return null

    // 兼容 Markdown 代码块及模型偶尔附带的前后说明。
    const withoutFence = result
        .replace(/^```(?:json)?\s*/i, '')
        .replace(/\s*```$/i, '')
        .trim()
    const firstBrace = withoutFence.indexOf('{')
    const lastBrace = withoutFence.lastIndexOf('}')
    const jsonText = firstBrace >= 0 && lastBrace > firstBrace
        ? withoutFence.slice(firstBrace, lastBrace + 1)
        : withoutFence

    const parsed = JSON.parse(jsonText)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        return null
    }

    return {
        ...parsed,
        meanings: Array.isArray(parsed.meanings)
            ? parsed.meanings.map(meaning => ({
                ...meaning,
                definitions: Array.isArray(meaning?.definitions) ? meaning.definitions : [],
            }))
            : [],
    }
}

async function cachedTts(text, lang) {
    const key = `${lang}:${text}`
    if (ttsCache.has(key)) {
        const cached = ttsCache.get(key)
        // 重新插入以维护简单的 LRU 顺序。
        ttsCache.delete(key)
        ttsCache.set(key, cached)
        // 每次返回副本，避免 decodeAudioData() transfer ArrayBuffer 后缓存失效
        return cached.slice()
    }
    const data = await lingva_tts.tts(text, lang)
    if (data) {
        if (ttsCache.size >= TTS_CACHE_LIMIT) {
            ttsCache.delete(ttsCache.keys().next().value)
        }
        // 存入副本，原始 data 传给 decodeAudioData 后会 detach，不影响缓存
        ttsCache.set(key, data.slice())
    }
    return data
}


export default function ToolBar() {
    const [result, setResult] = useState("")
    const [resultStream, setResultStream] = useState("")
    const [resultMeaningsStream, setResultMeaningsStream] = useState("")
    const [queryText, setQueryText] = useState("") // 原始查询文本
    const [isWord, setIsWord] = useState(false) // 是否为单词
    const [wordDetails, setWordDetails] = useState(null) // LLM 返回的词典 JSON
    const [isWordLoading, setIsWordLoading] = useState(false) // 单词查询加载中
    const queryTextRef = useRef('')
    const streamBufferRef = useRef(''); // 流式缓冲区
    const [isLoading, setIsLoading] = useState(false)
    const [isCopied, setIsCopied] = useState(false)
    const [isPlaying, setIsPlaying] = useState(false)
    const [isPlayingEn, setIsPlayingEn] = useState(false) // 播放英文
    const [isPlayingZh, setIsPlayingZh] = useState(false) // 播放中文
    const [playingExample, setPlayingExample] = useState('') // 当前播放的例句文本
    const [isPinned, setIsPinned] = useState(false) // 是否固定窗口
    const [isAnimating, setIsAnimating] = useState(true) // 动画状态
    const [slideDirection, setSlideDirection] = useState('right-bottom') // 滑入方向
    const [mode, setMode] = useState('translate') // 模式：translate/explain
    const modeRef = useRef('translate') // 用于在事件处理函数中访问最新的 mode 值
    const [explainTemplates, setExplainTemplates] = useState([]) // 解释模板列表
    const [selectedTemplate, setSelectedTemplate] = useState('') // 选中的模板ID
    const selectedTemplateRef = useRef('') // 用于在事件处理函数中访问最新的 selectedTemplate 值
    const [defaultTemplate, setDefaultTemplate] = useState('') // 默认模板ID
    const [currentTheme, setCurrentTheme] = useState(() => getSavedThemeId()) // 当前主题
    const playOrStop = useVoice()
    const contentRef = useRef(); // 实际内容容器的引用
    const resizeFrameRef = useRef(0)
    const lastRequestedHeightRef = useRef(0)
    const { t } = useTranslation(); // 国际化
    const theme = themes[currentTheme] || themes.warm; // 当前主题对象
    const copyText = useMemo(
        () => wordDetails?.translation || resultMeaningsStream || resultStream || result || '',
        [wordDetails, resultMeaningsStream, resultStream, result],
    )

    // 初始化主题
    useEffect(() => {
        applyThemeCssVars(currentTheme)
    }, [currentTheme])

    // 切换主题
    const handleThemeChange = useCallback((themeId) => {
        setCurrentTheme(themeId)
        saveThemeId(themeId)
        applyThemeCssVars(themeId)
    }, [])

    // 初始化时从后端获取固定状态、模式和模板列表
    useEffect(() => {
        GetToolBarPinned().then(pinned => {
            setIsPinned(pinned)
            // 如果已固定，设置窗口为始终置顶
            if (pinned) {
                Window.SetAlwaysOnTop(true)
            }
        }).catch(err => {
            console.error('获取固定状态失败:', err)
        })

        // 获取工具栏模式
        GetToolbarMode().then(mode => {
            if (mode) {
                setMode(mode)
                modeRef.current = mode
            }
        }).catch(err => {
            console.error('获取工具栏模式失败:', err)
        })

        // 获取解释模板列表
        GetExplainTemplates().then(result => {
            try {
                const data = JSON.parse(result)

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

        // 监听后端推送的模式更新
        const unsubscribeModeUpdated = Events.On("toolbarModeUpdated", function (data) {
            const newMode = typeof data.data === 'string' ? data.data : String(data.data || '')
            if (newMode) {
                setMode(newMode)
                modeRef.current = newMode
            }
        })

        // 监听后端推送的固定状态更新
        const unsubscribePinnedUpdated = Events.On("toolbarPinnedUpdated", function (data) {
            const pinned = !!data.data
            setIsPinned(pinned)
        })

        // 监听后端推送的弹窗滑入方向
        const unsubscribeSlideDirection = Events.On("toolbarSlideDirection", function (data) {
            const dir = typeof data.data === 'string' ? data.data : 'right-bottom'
            setSlideDirection(dir)
        })

        return () => {
            if (unsubscribeModeUpdated) unsubscribeModeUpdated()
            if (unsubscribePinnedUpdated) unsubscribePinnedUpdated()
            if (unsubscribeSlideDirection) unsubscribeSlideDirection()
        }
    }, [])

    // 检测是否为单个单词
    const checkIsWord = useCallback((text) => {
        if (!text) return false
        // 确保 text 是字符串类型
        const str = typeof text === 'string' ? text : String(text)
        const trimmed = str.trim()
        // 与后端保持一致：首尾为字母，中间可包含连字符或撇号，最长 30 字符。
        return CONSTANTS.WORD_REGEX.test(trimmed)
    }, [])



    // 复制到剪贴板
    const handleCopy = async () => {
        if (!copyText) return

        try {
            await navigator.clipboard.writeText(copyText)
            setIsCopied(true)
            setTimeout(() => setIsCopied(false), CONSTANTS.COPY_RESET_DELAY)
        } catch (err) {
            console.error('复制失败:', err)
        }
    }

    // 固定/取消固定窗口
    const handlePinToggle = async () => {
        const newPinnedState = !isPinned
        try {
            // 后端 SetToolBarPinned 会同时调用 Toolbar.SetAlwaysOnTop
            await SetToolBarPinned(newPinnedState)
            setIsPinned(newPinnedState)
        } catch (err) {
            console.error('设置固定状态失败:', err)
        }
    }


    // 播放英文单词发音
    const handleSpeakEnglish = async () => {
        if (!queryText) return
        try {
            setIsPlayingEn(true)
            const bytes = await cachedTts(queryText, 'en')
            if (bytes) await playOrStop(bytes)
        } catch (err) {
            console.error('英文播放失败:', err)
        } finally {
            setTimeout(() => setIsPlayingEn(false), CONSTANTS.PLAYING_RESET_DELAY)
        }
    }

    // 播放中文翻译发音（支持单词词典/普通翻译）
    const handleSpeakChinese = async () => {
        const text = wordDetails?.translation || result || resultStream
        if (!text) return
        try {
            setIsPlayingZh(true)
            const bytes = await cachedTts(text, 'zh')
            if (bytes) await playOrStop(bytes)
        } catch (err) {
            console.error('中文播放失败:', err)
        } finally {
            setTimeout(() => setIsPlayingZh(false), CONSTANTS.PLAYING_RESET_DELAY)
        }
    }

    // 语音播放（普通模式）
    const handleSpeak = async () => {
        if (isPlaying) {
            await playOrStop()
            setIsPlaying(false)
            return
        }
        if (!result && !resultStream) return
        try {
            setIsPlaying(true)
            const textToSpeak = result || resultStream
            const lang = /[\u4e00-\u9fa5]/.test(textToSpeak) ? 'zh' : 'en'
            const bytes = await cachedTts(textToSpeak, lang)
            if (bytes) await playOrStop(bytes)
        } catch (err) {
            console.error('语音播放失败:', err)
        } finally {
            setTimeout(() => setIsPlaying(false), CONSTANTS.PLAYING_RESET_DELAY)
        }
    }

    useEffect(() => {
        // 监听普通翻译结果
        const unsubscribeResult = Events.On("result", function (data) {
            // 确保 result 是字符串类型
            let result = typeof data.data === 'string' ? data.data : String(data.data || '')
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
            queryTextRef.current = text

            // ✅ 立即设置加载状态，防止窗口被隐藏
            setIsLoading(true)

            // 仅当查询文本变化时才清空词典结果（避免缓存命中时竞态）
            setQueryText(prev => {
                if (prev !== text) {
                    setWordDetails(null)
                }
                return text
            })
            streamBufferRef.current = '' // 重置流式缓冲区
            setResult('') // 清空显示
            setResultStream('') // 清空流式缓冲区
            setResultMeaningsStream("")

            // 检测是否为单词
            const isWordCheck = checkIsWord(text)
            setIsWord(isWordCheck)

            // 如果是单词，后端 processCurrentQuery 会直接调用 QueryWord
            // 结果通过 word_query_result 事件返回，前端只需设置加载状态
            if (isWordCheck && modeRef.current === 'translate') {
                setIsLoading(false)
                setIsWordLoading(true)
                // 不需要前端调用 QueryWord，后端自动处理
            }

            // 后端会根据保存的模式自动处理translate/explain，无需前端调用
        })

        // 监听流式翻译结果（后端发送的是累积的完整文本）
        const unsubscribeStream = Events.On("result_stream", function (data) {
            // 确保是字符串类型
            let fullText = typeof data.data === 'string' ? data.data : String(data.data || '')

            // ✅ 收到第一个数据时，清除加载状态
            if (fullText && streamBufferRef.current.length === 0) {
                setIsLoading(false)
            }

            streamBufferRef.current = fullText // 直接替换（不再追加）
            setResultStream(fullText) // 更新显示
        })
        const unsubscribeMeaningsStream = Events.On("result_meanings_stream", function (data) {
            let fullText = typeof data.data === 'string' ? data.data : String(data.data || '')

            if (fullText && streamBufferRef.current.length === 0) {
                setIsLoading(false)
            }

            streamBufferRef.current = fullText
            setResultMeaningsStream(fullText)
        })

        // 监听流式完成
        const unsubscribeStreamDone = Events.On("result_stream_done", function (data) {
            // ✅ 确保清除加载状态
            setIsLoading(false)
        })

        // 监听单词查询结果事件
        const unsubscribeWordQuery = Events.On("word_query_result", function (data) {
            const rawResult = typeof data.data === 'string' ? data.data : String(data.data || '')
            try {
                const parsed = parseWordDetails(rawResult)
                if (parsed) {
                    setWordDetails(parsed)
                } else {
                    throw new Error('单词查询结果不是 JSON 对象')
                }
            } catch (err) {
                console.error('解析单词查询结果失败:', err, '原始数据:', rawResult.substring(0, 200))
                // 非流式翻译源可能只返回纯文本。保留卡片布局，避免退化成单行弹窗。
                setWordDetails({
                    word: queryTextRef.current,
                    translation: rawResult,
                    meanings: [],
                })
            }
            setIsWordLoading(false)
        })

        // 清理事件监听
        return () => {
            if (unsubscribeResult) unsubscribeResult()
            if (unsubscribeQuery) unsubscribeQuery()
            if (unsubscribeStream) unsubscribeStream()
            if (unsubscribeStreamDone) unsubscribeStreamDone()
            if (unsubscribeMeaningsStream) unsubscribeMeaningsStream()
            if (unsubscribeWordQuery) unsubscribeWordQuery()
        }
    }, [])

    useEffect(() => {
        // 检查是否有内容或正在加载
        const hasContent = !!(result || resultStream || resultMeaningsStream || wordDetails || isLoading || isWordLoading)

        if (!hasContent) {
            lastRequestedHeightRef.current = 0
            // 无内容且未加载时隐藏窗口
            const timer = setTimeout(() => {
                Hide("ToolBar").catch(() => {
                })
            }, 100)
            return () => clearTimeout(timer)
        }

        // 如果正在加载，显示固定高度的加载窗口
        if (isLoading) {
            const loadingHeight = CONSTANTS.LOADING_HEIGHT
            lastRequestedHeightRef.current = loadingHeight
            ToolBarShow(loadingHeight)
            return
        }

        const content = contentRef.current
        if (!content) return

        const updateToolbarHeight = () => {
            cancelAnimationFrame(resizeFrameRef.current)
            resizeFrameRef.current = requestAnimationFrame(() => {
                if (!contentRef.current) return

                const measuredHeight = Math.ceil(Math.max(
                    contentRef.current.scrollHeight,
                    contentRef.current.getBoundingClientRect().height,
                ))
                const minimumHeight = isWord && mode !== 'explain' ? 150 : CONSTANTS.LOADING_HEIGHT
                const actualContentHeight = Math.min(
                    Math.max(measuredHeight, minimumHeight),
                    CONSTANTS.MAX_CONTENT_HEIGHT,
                )

                if (lastRequestedHeightRef.current !== actualContentHeight) {
                    lastRequestedHeightRef.current = actualContentHeight
                    ToolBarShow(actualContentHeight)
                }
            })
        }

        // 单词卡片包含异步结果和第三方 UI 组件，实际布局可能晚于 React commit。
        // 持续观察尺寸，避免窗口停留在快速翻译的一行高度。
        const resizeObserver = new ResizeObserver(updateToolbarHeight)
        resizeObserver.observe(content)
        updateToolbarHeight()

        return () => {
            resizeObserver.disconnect()
            cancelAnimationFrame(resizeFrameRef.current)
        }
    }, [result, resultStream, resultMeaningsStream, isWord, wordDetails, isLoading, isWordLoading, mode]);

    // 获取词性标签样式
    const getPartOfSpeechStyle = (partOfSpeech) => {
        const styles = {
            'noun': 'bg-[#FFF5E6] text-[#92400E]',        // 琥珀色系
            'verb': 'bg-[#FEF7E0] text-[#78350F]',        // 深金色系
            'adjective': 'bg-[#FFF0F0] text-[#9A3412]',   // 暖红色系
            'adverb': 'bg-[#FFF8F0] text-[#A16207]',      // 橙棕色系
            'pronoun': 'bg-[#FFF5F5] text-[#C4533A]',     // 赤陶色系
            'preposition': 'bg-[#FEFCE8] text-[#854D0E]', // 土金色系
            'conjunction': 'bg-[#FDF5EF] text-[#6B4226]', // 棕色系
            'interjection': 'bg-[#FEF2F2] text-[#B91C1C]', // 暖赤色
        }
        return styles[partOfSpeech] || 'bg-[#F5F0E8] text-[#6B5744]'
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
            'plural': 'pl.',
        }
        return (abbrs[partOfSpeech] || partOfSpeech + '.').toUpperCase()
    }




    // 播放例句发音
    const handleSpeakExample = async (exampleText) => {
        if (!exampleText) return
        try {
            setPlayingExample(exampleText)
            const bytes = await cachedTts(exampleText, 'en')
            if (bytes) await playOrStop(bytes)
        } catch (err) {
            console.error('播放例句失败:', err)
        } finally {
            setPlayingExample('')
        }
    }

    // 将单词在例句中高亮
    const highlightWord = (text, word) => {
        if (!text || !word) return text
        const parts = []
        let lastIndex = 0
        let match
        const regexGlobal = new RegExp(`\\b(${word}[a-z]*)\\b`, 'gi')
        while ((match = regexGlobal.exec(text)) !== null) {
            if (match.index > lastIndex) {
                parts.push(text.substring(lastIndex, match.index))
            }
            parts.push(
                <span key={match.index} className="font-[700] rounded-[2px] px-[2px] mx-[1px] not-italic transition-all duration-200" style={{ color: 'var(--color-highlight-text)', backgroundColor: 'var(--color-highlight-bg)' }}>
                    {match[0]}
                </span>
            )
            lastIndex = match.index + match[0].length
        }

        if (lastIndex < text.length) {
            parts.push(text.substring(lastIndex))
        }
        return parts.length > 0 ? parts : text
    }

    // 渲染加载动画
    const renderLoading = () => {
        return (
            <div className="loading-container p-3 flex flex-col items-center justify-center space-y-2 min-h-[60px]">
                <div className="space-y-2.5 w-full">
                    <div className="h-3 w-4/5 rounded-full loading-skeleton"></div>
                    <div className="h-3 w-full rounded-full loading-skeleton" style={{ animationDelay: '0.15s' }}></div>
                    <div className="h-3 w-3/5 rounded-full loading-skeleton" style={{ animationDelay: '0.3s' }}></div>
                </div>
            </div>
        )
    }

    // 渲染词典格式内容
    const renderWordDetailsContent = () => {

        return (
            <>
                {/* 单词 + 音标 + 播放按钮 */}
                <div className="flex items-center gap-[10px] mb-[10px]">
                    <h3 className="text-[22px] font-[600] tracking-tight leading-none" style={{ color: 'var(--color-text-main)' }}>{queryText}</h3>
                    {wordDetails?.phonetic && (
                        <span className="text-[14px] font-normal tracking-wide mt-[4px]" style={{ color: 'var(--color-text-secondary)' }}>/{wordDetails.phonetic.replace(/[\/\[\]]/g, '')}/</span>
                    )}
                    <Tooltip content="播放发音" placement="top" delay={500}>
                        <Button
                            size="sm"
                            isIconOnly
                            variant="flat"
                            radius="full"
                            aria-label="Play English"
                            onPress={handleSpeakEnglish}
                            isLoading={isPlayingEn}
                            className={`shrink-0 w-[30px] h-[30px] min-w-[30px] bg-transparent ${theme.classes.speakBtn} active:scale-95 transition-all duration-300 mt-[2px]`}
                        >
                            <MdVolumeUp className="text-[20px]" />
                        </Button>
                    </Tooltip>
                </div>

                {/* 词性和释义 */}
                {wordDetails?.meanings && wordDetails.meanings.length > 0 ? (
                    wordDetails.meanings.map((meaning, idx) => (
                        <div key={idx} className={`word-card-section ${idx > 0 ? 'mt-[10px] pt-[10px]' : ''}`} style={idx > 0 ? { borderTop: '1px solid var(--color-border-light)' } : undefined}>
                            {/* 词性标签 */}
                            <span className={`inline-block px-[8px] py-[2px] rounded-[4px] text-[11px] font-[600] uppercase tracking-wider mb-2 ${getPartOfSpeechStyle(meaning.partOfSpeech)}`}>
                                {getPartOfSpeechAbbr(meaning.partOfSpeech)}
                            </span>

                            {/* 释义列表 */}
                            <div className="space-y-[10px]">
                                {meaning.definitions.map((def, defIdx) => {
                                    return (
                                        <div key={defIdx}>
                                            {/* 英文释义 + 中文翻译 */}
                                            <div className="flex flex-col gap-[4px] mb-[8px]">
                                                <p className="text-[13px] leading-[1.6]" style={{ color: 'var(--color-text-main)' }}>{def.definition}</p>
                                                {def.definitionZh && (
                                                    <p className="text-[13px] leading-[1.6]" style={{ color: 'var(--color-body)' }}>{def.definitionZh}</p>
                                                )}
                                            </div>

                                            {/* 英文例句 */}
                                            {def.example && (
                                                <div className="px-[10px] py-[6px] rounded-r-[6px]" style={{ background: 'var(--color-example-bg)', borderLeft: '2px solid var(--color-example-border)' }}>
                                                    <div className="flex items-start gap-3">
                                                        <p className="flex-1 text-[12.5px] leading-[1.6]" style={{ color: 'var(--color-text-main)' }}>
                                                            "{highlightWord(def.example, queryText)}"
                                                        </p>
                                                        <Tooltip content="播放例句" placement="top" delay={500}>
                                                            <Button
                                                                size="sm"
                                                                isIconOnly
                                                                variant="flat"
                                                                radius="full"
                                                                aria-label="Play Example"
                                                                onPress={() => handleSpeakExample(def.example)}
                                                                isLoading={playingExample === def.example}
                                                                className={`shrink-0 w-[28px] h-[28px] min-w-[28px] ${theme.classes.speakBtnExample} active:scale-95 transition-all duration-300`}
                                                            >
                                                                <MdVolumeUp className="text-[16px]" />
                                                            </Button>
                                                        </Tooltip>
                                                    </div>
                                                    {def.exampleZh && (
                                                        <p className="mt-[4px] text-[12px] leading-[1.5]" style={{ color: 'var(--color-muted)' }}>{def.exampleZh}</p>
                                                    )}
                                                </div>
                                            )}
                                        </div>
                                    )
                                })}
                            </div>
                        </div>
                    ))

                ) : isWordLoading ? (
                    // 单词查询加载中，显示骨架屏
                    <div className="space-y-3 mb-2">
                        <div className="flex items-center gap-2">
                            <div className="h-4 w-16 rounded loading-skeleton"></div>
                            <div className="h-3 w-24 rounded loading-skeleton" style={{ animationDelay: '0.1s' }}></div>
                        </div>
                        <div className="space-y-2">
                            <div className="h-3 w-full rounded loading-skeleton" style={{ animationDelay: '0.15s' }}></div>
                            <div className="h-3 w-4/5 rounded loading-skeleton" style={{ animationDelay: '0.25s' }}></div>
                            <div className="h-3 w-3/5 rounded loading-skeleton" style={{ animationDelay: '0.35s' }}></div>
                        </div>
                    </div>
                ) : (
                    // 如果没有词典数据，显示提示
                    <div className="mb-2 text-sm text-gray-500">
                        词典暂无详细释义
                    </div>
                )}

                {/* 底部中文翻译 */}
                {(wordDetails?.translation || resultStream) && (
                    <div className="pt-[8px] mt-[10px] flex items-center justify-between gap-2" style={{ borderTop: '1px solid var(--color-border-section)' }}>
                        <p className="text-[14px] font-[500] leading-[1.5] flex-1" style={{ color: 'var(--color-text-main)' }}>{wordDetails?.translation || resultStream}</p>
                        <Tooltip content="播放发音" placement="top" delay={500}>
                            <Button
                                size="sm"
                                isIconOnly
                                variant="flat"
                                radius="full"
                                aria-label="Play Chinese"
                                onPress={handleSpeakChinese}
                                isLoading={isPlayingZh}
                                className={`shrink-0 w-[30px] h-[30px] min-w-[30px] bg-transparent ${theme.classes.speakBtn} active:scale-95 transition-all duration-300`}
                            >
                                <MdVolumeUp className="text-[20px]" />
                            </Button>
                        </Tooltip>
                    </div>
                )}
            </>
        )

    }

    // 根据滑入方向生成动画 class
    const slideAnimClass = `toolbar-slide-${slideDirection}`

    return (
        <div className={slideAnimClass}>
        <Card
            shadow="none"
            className='rounded-[18px] w-full bg-white border border-black/[0.04] transition-all duration-200 overflow-hidden'
            style={{ boxShadow: '0 1px 3px rgba(0,0,0,0.08)' }}>

            <CardHeader className={`px-[12px] py-[6px] ${theme.classes.header} flex justify-between items-center`} style={{ "--wails-draggable": "drag" }}>

                <div className="flex gap-[6px] items-center w-full justify-between" style={{ WebkitAppRegion: 'drag' }}>
                    {/* 应用 Logo 面板锚点 */}
                    <div className={`flex justify-center items-center w-[26px] h-[26px] min-w-[26px] select-none pointer-events-none ${theme.classes.logoBg} rounded-full shadow-sm`}>
                        <img src="/appicon.png" alt="Handy Translate" className="w-[16px] h-[16px] object-contain drop-shadow-sm" draggable="false" />
                    </div>

                    {/* 翻译/解释模式切换 */}
                    <Tabs
                        selectedKey={mode}
                        onSelectionChange={async (key) => {
                            setMode(key)
                            modeRef.current = key // 同步更新 ref
                            // 通知后端更新模式（后端会自动保存并处理）
                            Events.Emit({ name: "toolbarMode", data: key })

                            // 切换模式后，如果有queryText，重新调用对应API
                            if (queryText && queryText.trim() !== '') {
                                setIsLoading(true)
                                setResult('')
                                setResultStream('')
                                setResultMeaningsStream('')
                                streamBufferRef.current = ''
                                if (key === 'explain') {
                                    setWordDetails(null)
                                }
                            }
                        }}
                        size="sm"
                        aria-label="Mode Switch"
                        classNames={{
                            tabList: theme.classes.tabList,
                            cursor: theme.classes.tabCursor,
                            tab: "h-[24px] px-[10px] rounded-[8px] transition-all duration-300",
                            tabContent: theme.classes.tabContent
                        }}
                    >
                        <Tab
                            key="translate"
                            title={
                                <div className="flex items-center gap-[6px]">
                                    <BsTranslate className="text-[12px]" />
                                    <span className="text-[12px]">{t('translate.translate')}</span>
                                </div>
                            }
                        />
                        <Tab
                            key="explain"
                            title={
                                <div className="flex items-center gap-[6px]">
                                    <MdLightbulb className="text-[14px]" />
                                    <span className="text-[12px]">{t('translate.explain')}</span>
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
                                    variant="light"
                                    endContent={<MdKeyboardArrowDown className="text-[14px] opacity-55 shrink-0" />}
                                    className="h-[26px] min-w-0 max-w-[104px] px-[9px] gap-[3px] rounded-full bg-white/55 hover:bg-white/80 border border-white/70 text-[11px] font-medium shadow-none"
                                    style={{ WebkitAppRegion: 'no-drag' }}
                                >
                                    <span className="truncate max-w-[76px]">
                                        {explainTemplates.find(t => t.id === selectedTemplate)?.name || explainTemplates.find(t => t.id === defaultTemplate)?.name || t('translate.template_placeholder')}
                                    </span>
                                </Button>
                            </DropdownTrigger>
                            <DropdownMenu
                                aria-label={t('translate.select_template')}
                                selectedKeys={selectedTemplate ? [selectedTemplate] : []}
                                selectionMode="single"
                                onAction={async (key) => {
                                    const newTemplateId = String(key)
                                    setSelectedTemplate(newTemplateId)
                                    selectedTemplateRef.current = newTemplateId // 同步更新 ref

                                    // 更新默认模板到后端（这样下次查询时会使用新模板）
                                    try {
                                        await SetDefaultExplainTemplate(newTemplateId)
                                    } catch (err) {
                                        console.error('更新默认模板失败:', err)
                                    }

                                    // 切换模板后，如果有查询文本，清空结果等待下次查询
                                    // 注意：当前查询不会立即使用新模板，需要重新触发查询
                                    if (queryText && queryText.trim() !== '') {
                                        setResult('')
                                        setResultStream('')
                                        setResultMeaningsStream('')
                                        streamBufferRef.current = ''
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

                    <div className="flex gap-[6px]">
                        <Tooltip content={isPinned ? "取消固定" : "固定窗口"} placement="bottom" delay={500}>
                            <Button
                                size="sm"
                                isIconOnly
                                variant="flat"
                                className={`w-[28px] h-[28px] min-w-[28px] rounded-full active:scale-90 transition-all duration-300 ${isPinned ? theme.classes.pinActive : theme.classes.pinInactive}`}
                                aria-label="Pin"
                                onPress={handlePinToggle}
                            >
                                {isPinned ? <MdPushPin className="text-[16px] drop-shadow-sm" /> : <MdOutlinePushPin className="text-[16px]" />}
                            </Button>
                        </Tooltip>

                        <Tooltip content={isCopied ? "已复制!" : "复制"} placement="bottom" delay={500}>
                            <Button
                                size="sm"
                                isIconOnly
                                variant="flat"
                                className={`w-[28px] h-[28px] min-w-[28px] rounded-full active:scale-90 transition-all duration-300 ${isCopied ? theme.classes.copyActive : theme.classes.copyInactive}`}
                                aria-label="Copy"
                                onPress={handleCopy}
                                isDisabled={!copyText}
                            >
                                {isCopied ? <MdCheck className="text-[16px] drop-shadow-sm animate-appearance-in" /> : <MdContentCopy className="text-[16px]" />}
                            </Button>
                        </Tooltip>

                        {/* 主题切换 */}
                        <Popover placement="bottom" offset={8}>
                            <PopoverTrigger>
                                <Button
                                    size="sm"
                                    isIconOnly
                                    variant="flat"
                                    className={`w-[28px] h-[28px] min-w-[28px] rounded-full active:scale-90 transition-all duration-300 ${theme.classes.themeBtn}`}
                                    aria-label="Theme"
                                >
                                    <MdPalette className="text-[16px]" />
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent className="p-3 bg-white/95 backdrop-blur-md shadow-lg rounded-[14px] border border-black/5">
                                <div className="flex gap-[10px] items-center">
                                    {themeIds.map(id => (
                                        <Tooltip key={id} content={themes[id].name} placement="bottom" delay={300}>
                                            <button
                                                onClick={() => handleThemeChange(id)}
                                                className="relative w-[28px] h-[28px] rounded-full transition-all duration-300 hover:scale-110 active:scale-95 focus:outline-none"
                                                style={{
                                                    background: themes[id].dot,
                                                    boxShadow: currentTheme === id
                                                        ? `0 0 0 2.5px white, 0 0 0 4.5px ${themes[id].dot}`
                                                        : '0 1px 3px rgba(0,0,0,0.15)',
                                                }}
                                                aria-label={themes[id].name}
                                            >
                                                {currentTheme === id && (
                                                    <MdCheck className="absolute inset-0 m-auto text-white text-[16px] drop-shadow-sm" />
                                                )}
                                            </button>
                                        </Tooltip>
                                    ))}
                                </div>
                            </PopoverContent>
                        </Popover>

                    </div>


                </div>
            </CardHeader>


            <CardBody className="overflow-y-auto overflow-x-hidden px-[12px] pb-[6px] pt-0" style={{ maxHeight: 'calc(100vh - 42px)' }}>
                {isLoading && !resultStream && !isWordLoading ? (
                    renderLoading()
                ) : !(result || resultStream || resultMeaningsStream || (isWord && (wordDetails || isWordLoading))) ? (
                    <div ref={contentRef} className="empty-state">
                        <MdSearch className="empty-state-icon" />
                        <span>选中文字后点击鼠标中键</span>
                    </div>
                ) : (
                    <div ref={contentRef} className="w-full">
                        {isWord && mode !== 'explain' ? (
                            <div className="w-full">
                                {renderWordDetailsContent()}
                            </div>

                        ) : mode === 'explain' ? (
                            <div className="markdown-content leading-relaxed pb-0">
                                <ReactMarkdown
                                    components={{
                                        h1: ({ node, ...props }) => <h1 className="text-xl font-bold mb-3 mt-4" style={{ color: 'var(--color-heading)' }} {...props} />,
                                        h2: ({ node, ...props }) => <h2 className="text-lg font-bold mb-2 mt-3" style={{ color: 'var(--color-heading)' }} {...props} />,
                                        h3: ({ node, ...props }) => <h3 className="text-base font-semibold mb-2 mt-3" style={{ color: 'var(--color-heading)' }} {...props} />,
                                        h4: ({ node, ...props }) => <h4 className="text-sm font-semibold mb-1 mt-2" style={{ color: 'var(--color-body)' }} {...props} />,
                                        p: ({ node, ...props }) => <p className="mb-3 leading-relaxed break-words" style={{ wordBreak: 'break-word', color: 'var(--color-body)' }} {...props} />,
                                        ul: ({ node, ...props }) => <ul className="list-disc list-inside mb-3 space-y-1 ml-4" style={{ color: 'var(--color-body)' }} {...props} />,
                                        ol: ({ node, ...props }) => <ol className="list-decimal list-inside mb-3 space-y-1 ml-4" style={{ color: 'var(--color-body)' }} {...props} />,
                                        li: ({ node, ...props }) => <li className="leading-relaxed" {...props} />,
                                        code: ({ node, inline, ...props }) =>
                                            inline ? (
                                                <code className="px-1.5 py-0.5 rounded text-sm font-mono" style={{ background: 'var(--color-code-bg)', color: 'var(--color-code-text)', border: '1px solid var(--color-code-border)' }} {...props} />
                                            ) : (
                                                <code className="block p-3 rounded-lg mb-3 overflow-x-auto font-mono text-sm whitespace-pre" style={{ background: 'var(--color-pre-bg)', color: 'var(--color-body)', border: '1px solid var(--color-pre-border)' }} {...props} />
                                            ),
                                        pre: ({ node, ...props }) => <pre className="rounded-lg mb-3 overflow-x-auto" style={{ background: 'var(--color-pre-bg)', border: '1px solid var(--color-pre-border)' }} {...props} />,
                                        blockquote: ({ node, ...props }) => <blockquote className="pl-4 italic my-3" style={{ borderLeft: '3px solid var(--color-quote-border)', color: 'var(--color-muted)' }} {...props} />,
                                        strong: ({ node, ...props }) => <strong className="font-semibold" style={{ color: 'var(--color-heading)' }} {...props} />,
                                        em: ({ node, ...props }) => <em className="italic" style={{ color: 'var(--color-muted)' }} {...props} />,
                                        hr: ({ node, ...props }) => <hr className="my-4" style={{ borderColor: 'var(--color-border-section)' }} {...props} />,
                                    }}
                                >
                                    {resultStream || result}
                                </ReactMarkdown>
                            </div>
                        ) : (
                            <p className="result-text whitespace-pre-wrap pt-2 pb-0">
                                <span className={resultStream && !result ? 'typing-cursor' : ''}>
                                    {resultStream || result}
                                </span>
                            </p>
                        )}
                    </div>
                )}
            </CardBody>

        </Card >
        </div>
    );
}
