//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	guiClassName = "AgentFeishuWindow"

	idConnect   = 1001
	idReceiver  = 1002
	idAgent     = 1003
	idProject   = 1010
	idTarget    = 1011
	idBrowse    = 1012
	idAdd       = 1013
	idList      = 1014
	idHide      = 1015
	idExit      = 1016
	idStatus    = 1017
	idAutostart = 1018
	idTest      = 1019

	wsOverlappedWindow  = 0x00CF0000
	wsThickFrame        = 0x00040000
	wsMaximizeBox       = 0x00010000
	wsChild             = 0x40000000
	wsVisible           = 0x10000000
	wsBorder            = 0x00800000
	wsVScroll           = 0x00200000
	wsTabStop           = 0x00010000
	wsClipSiblings      = 0x04000000
	esLeft              = 0x0000
	esMultiline         = 0x0004
	esPassword          = 0x0020
	esAutoVScroll       = 0x0040
	esReadOnly          = 0x0800
	bsPushButton        = 0x00000000
	bsAutoCheckBox      = 0x00000003
	bsFlat              = 0x00008000
	cbsDropdownList     = 0x0003
	lbsNotify           = 0x00000001
	lbsOwnerDrawFixed   = 0x00000010
	lbsHasStrings       = 0x00000040
	lbsNoIntegralHeight = 0x00000100

	cwUseDefault = 0x80000000
	swShow       = 5
	swHide       = 0
	sbVert       = 1

	wmDestroy         = 0x0002
	wmPaint           = 0x000F
	wmClose           = 0x0010
	wmEraseBkgnd      = 0x0014
	wmMeasureItem     = 0x002C
	wmDrawItem        = 0x002B
	wmCommand         = 0x0111
	wmAppTray         = 0x8001
	wmUser            = 0x0400
	wmAppRegisterDone = wmUser + 1
	wmAppQRCodeReady  = wmUser + 2
	wmAppTestDone     = wmUser + 3
	wmSetFont         = 0x0030
	wmLButtonD        = 0x0203
	wmSize            = 0x0005
	wmVScroll         = 0x0115
	wmMouseWheel      = 0x020A

	sbLineUp        = 0
	sbLineDown      = 1
	sbPageUp        = 2
	sbPageDown      = 3
	sbThumbPosition = 4
	sbThumbTrack    = 5
	sbTop           = 6
	sbBottom        = 7

	sifRange    = 0x0001
	sifPage     = 0x0002
	sifPos      = 0x0004
	sifTrackPos = 0x0010

	spiGetWorkArea = 0x0030

	guiContentHeight = 960

	bmGetCheck = 0x00F0
	bmSetCheck = 0x00F1
	bstChecked = 0x0001

	cbAddString     = 0x0143
	cbSetCurSel     = 0x014E
	cbGetCurSel     = 0x0147
	lbAddString     = 0x0180
	lbReset         = 0x0184
	lbGetText       = 0x0189
	lbGetTextLen    = 0x018A
	lbSetItemHeight = 0x01A0

	odsSelected = 0x0001
	psSolid     = 0
	transparent = 1

	dtLeft        = 0x00000000
	dtVCenter     = 0x00000004
	dtSingleLine  = 0x00000020
	dtNoPrefix    = 0x00000800
	dtEndEllipsis = 0x00008000

	nimAdd     = 0x00000000
	nimDelete  = 0x00000002
	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	coinitApartmentThreaded = 0x00000002

	hkeyCurrentUser = 0x80000001
	keyQueryValue   = 0x0001
	keySetValue     = 0x0002
	regSz           = 0x00000001
	errFileNotFound = 2

	startupRunKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupValueName = "Agent Feishu"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")

	pRegisterClassEx     = user32.NewProc("RegisterClassExW")
	pCreateWindowEx      = user32.NewProc("CreateWindowExW")
	pDefWindowProc       = user32.NewProc("DefWindowProcW")
	pGetClientRect       = user32.NewProc("GetClientRect")
	pShowWindow          = user32.NewProc("ShowWindow")
	pShowScrollBar       = user32.NewProc("ShowScrollBar")
	pMoveWindow          = user32.NewProc("MoveWindow")
	pSetScrollInfo       = user32.NewProc("SetScrollInfo")
	pGetScrollInfo       = user32.NewProc("GetScrollInfo")
	pSystemParameters    = user32.NewProc("SystemParametersInfoW")
	pSetViewportOrgEx    = gdi32.NewProc("SetViewportOrgEx")
	pUpdateWindow        = user32.NewProc("UpdateWindow")
	pGetMessage          = user32.NewProc("GetMessageW")
	pTranslateMessage    = user32.NewProc("TranslateMessage")
	pDispatchMessage     = user32.NewProc("DispatchMessageW")
	pPostMessage         = user32.NewProc("PostMessageW")
	pPostQuitMessage     = user32.NewProc("PostQuitMessage")
	pBeginPaint          = user32.NewProc("BeginPaint")
	pEndPaint            = user32.NewProc("EndPaint")
	pFillRect            = user32.NewProc("FillRect")
	pInvalidateRect      = user32.NewProc("InvalidateRect")
	pDrawText            = user32.NewProc("DrawTextW")
	pGetWindowText       = user32.NewProc("GetWindowTextW")
	pGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	pSetWindowText       = user32.NewProc("SetWindowTextW")
	pSendMessage         = user32.NewProc("SendMessageW")
	pMessageBox          = user32.NewProc("MessageBoxW")
	pLoadIcon            = user32.NewProc("LoadIconW")
	pDestroyWindow       = user32.NewProc("DestroyWindow")
	pGetStockObject      = gdi32.NewProc("GetStockObject")
	pCreateFont          = gdi32.NewProc("CreateFontW")
	pCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	pCreatePen           = gdi32.NewProc("CreatePen")
	pRectangle           = gdi32.NewProc("Rectangle")
	pSelectObject        = gdi32.NewProc("SelectObject")
	pDeleteObject        = gdi32.NewProc("DeleteObject")
	pRoundRect           = gdi32.NewProc("RoundRect")
	pSetBkMode           = gdi32.NewProc("SetBkMode")
	pSetTextColor        = gdi32.NewProc("SetTextColor")
	pGetModuleHandle     = kernel32.NewProc("GetModuleHandleW")
	pShellNotifyIcon     = shell32.NewProc("Shell_NotifyIconW")
	pSHBrowseForFolder   = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDList = shell32.NewProc("SHGetPathFromIDListW")
	pCoInitializeEx      = ole32.NewProc("CoInitializeEx")
	pCoUninitialize      = ole32.NewProc("CoUninitialize")
	pCoTaskMemFree       = ole32.NewProc("CoTaskMemFree")
	pRegCreateKeyEx      = advapi32.NewProc("RegCreateKeyExW")
	pRegOpenKeyEx        = advapi32.NewProc("RegOpenKeyExW")
	pRegSetValueEx       = advapi32.NewProc("RegSetValueExW")
	pRegQueryValueEx     = advapi32.NewProc("RegQueryValueExW")
	pRegDeleteValue      = advapi32.NewProc("RegDeleteValueW")
	pRegCloseKey         = advapi32.NewProc("RegCloseKey")
)

type guiApp struct {
	hwnd         uintptr
	installDir   string
	installPath  string
	startHidden  bool
	controls     map[int]uintptr
	statusText   string
	font         uintptr
	titleFont    uintptr
	sectionFont  uintptr
	smallFont    uintptr
	cardFont     uintptr
	mu           sync.Mutex
	registering  bool
	lastAsyncErr error
	qrMatrix     [][]bool
	qrURL        string
	qrExpires    int
	qrErr        error
	scrollY      int32
	contentH     int32
	design       map[int]ctrlPos
}

var activeGUI *guiApp

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type point struct {
	x, y int32
}

type rect struct {
	left, top, right, bottom int32
}

type msg struct {
	hWnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type measureItemStruct struct {
	ctlType    uint32
	ctlID      uint32
	itemID     uint32
	itemWidth  uint32
	itemHeight uint32
	itemData   uintptr
}

type drawItemStruct struct {
	ctlType    uint32
	ctlID      uint32
	itemID     uint32
	itemAction uint32
	itemState  uint32
	hwndItem   uintptr
	hdc        uintptr
	rcItem     rect
	itemData   uintptr
}

type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

type browseInfo struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

type scrollInfo struct {
	cbSize    uint32
	fMask     uint32
	nMin      int32
	nMax      int32
	nPage     uint32
	nPos      int32
	nTrackPos int32
}

type ctrlPos struct {
	x, y, w, h int32
}

func runNativeGUI(args []string, stdout interface{}) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	guiLog("runNativeGUI: start args=%q", strings.Join(args, " "))
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	installDirFlag := fs.String("install-dir", "", "install directory")
	startHidden := fs.Bool("start-hidden", false, "start hidden in the system tray")
	if err := fs.Parse(args); err != nil {
		guiLog("runNativeGUI: parse error: %v", err)
		return err
	}
	installDir := firstNonEmpty(*installDirFlag, defaultInstallDir())
	installPath := filepath.Join(installDir, executableFileName())
	guiLog("runNativeGUI: installDir=%s installPath=%s", installDir, installPath)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		guiLog("runNativeGUI: mkdir error: %v", err)
		return err
	}
	if err := copySelfTo(installPath); err != nil {
		guiLog("runNativeGUI: copySelfTo error: %v", err)
		return err
	}
	guiLog("runNativeGUI: copySelfTo ok")
	if err := writeSnippet(filepath.Join(installDir, "AGENTS-snippet.md"), installPath); err != nil {
		guiLog("runNativeGUI: writeSnippet error: %v", err)
		return err
	}
	guiLog("runNativeGUI: writeSnippet ok")
	comInitialized := initializeCOM()
	if comInitialized {
		defer pCoUninitialize.Call()
	}

	app := &guiApp{
		installDir:  installDir,
		installPath: installPath,
		startHidden: *startHidden,
		controls:    map[int]uintptr{},
		design:      map[int]ctrlPos{},
		contentH:    guiContentHeight,
		statusText:  "就绪：先扫码连接飞书自建应用，再添加项目文件夹。",
	}
	activeGUI = app
	return app.run()
}

func (a *guiApp) run() error {
	guiLog("gui.run: start")
	instance, _, _ := pGetModuleHandle.Call(0)
	className := utf16Ptr(guiClassName)
	icon, _, _ := pLoadIcon.Call(0, uintptr(32512))
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   syscall.NewCallback(windowProc),
		hInstance:     instance,
		hIcon:         icon,
		hIconSm:       icon,
		hbrBackground: 16,
		lpszClassName: className,
	}
	if r, _, err := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		guiLog("gui.run: RegisterClassExW failed: %v", err)
		return fmt.Errorf("RegisterClassExW failed: %v", err)
	}
	guiLog("gui.run: RegisterClassExW ok")
	winW, winH := fitWindowSize(1160, 872)
	hwnd, _, err := pCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("Agent Feishu"))),
		wsOverlappedWindow&^(wsThickFrame|wsMaximizeBox)|wsVScroll,
		cwUseDefault, cwUseDefault, uintptr(winW), uintptr(winH),
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		guiLog("gui.run: CreateWindowExW failed: %v", err)
		return fmt.Errorf("CreateWindowExW failed: %v", err)
	}
	guiLog("gui.run: CreateWindowExW ok hwnd=%d", hwnd)
	a.hwnd = hwnd
	guiLog("gui.run: createControls start")
	a.createControls(instance)
	guiLog("gui.run: createControls ok")
	guiLog("gui.run: loadConfigToControls start")
	a.loadConfigToControls()
	guiLog("gui.run: loadConfigToControls ok")
	guiLog("gui.run: refreshProjectList start")
	a.refreshProjectList()
	guiLog("gui.run: refreshProjectList ok")
	guiLog("gui.run: addTrayIcon start")
	a.addTrayIcon()
	guiLog("gui.run: addTrayIcon ok")
	if a.startHidden {
		pShowWindow.Call(hwnd, swHide)
		guiLog("gui.run: ShowWindow hide")
	} else {
		pShowWindow.Call(hwnd, swShow)
		guiLog("gui.run: ShowWindow show")
	}
	pUpdateWindow.Call(hwnd)
	a.updateScroll()
	guiLog("gui.run: UpdateWindow ok, entering message loop")
	var m msg
	for {
		ret, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			guiLog("gui.run: GetMessage end ret=%d", ret)
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
	guiLog("gui.run: exit")
	return nil
}

func (a *guiApp) createControls(instance uintptr) {
	a.font = createFont(16, 400)
	a.titleFont = createFont(28, 700)
	a.sectionFont = createFont(18, 700)
	a.smallFont = createFont(14, 400)
	a.cardFont = createFont(19, 700)
	if a.font == 0 {
		a.font, _, _ = pGetStockObject.Call(17)
	}
	createWithFont := func(id int, class, text string, style uint32, x, y, w, h int32, useFont uintptr) uintptr {
		hwnd, _, _ := pCreateWindowEx.Call(
			0,
			uintptr(unsafe.Pointer(utf16Ptr(class))),
			uintptr(unsafe.Pointer(utf16Ptr(text))),
			uintptr(wsChild|wsVisible|wsClipSiblings|style),
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			a.hwnd, uintptr(id), instance, 0,
		)
		if hwnd != 0 && useFont != 0 {
			pSendMessage.Call(hwnd, wmSetFont, useFont, 1)
		}
		a.controls[id] = hwnd
		a.design[id] = ctrlPos{x, y, w, h}
		return hwnd
	}
	create := func(id int, class, text string, style uint32, x, y, w, h int32) uintptr {
		return createWithFont(id, class, text, style, x, y, w, h, a.font)
	}

	create(idHide, "BUTTON", "隐藏", bsPushButton|bsFlat|wsTabStop, 920, 28, 74, 34)
	create(idExit, "BUTTON", "退出", bsPushButton|bsFlat|wsTabStop, 1010, 28, 70, 34)

	create(idConnect, "BUTTON", "生成二维码", bsPushButton|bsFlat|wsTabStop, 820, 186, 150, 42)
	create(idTest, "BUTTON", "发送测试", bsPushButton|bsFlat|wsTabStop, 820, 232, 150, 38)
	create(idReceiver, "EDIT", "", esLeft|esReadOnly|wsTabStop, 780, 272, 240, 28)
	create(idAutostart, "BUTTON", "开机后常驻托盘", bsAutoCheckBox|wsTabStop, 780, 310, 200, 24)

	create(idProject, "EDIT", "", esLeft|wsTabStop, 520, 448, 390, 24)
	create(idBrowse, "BUTTON", "浏览...", bsPushButton|bsFlat|wsTabStop, 930, 438, 86, 40)
	target := create(idTarget, "COMBOBOX", "", wsBorder|cbsDropdownList|wsTabStop, 520, 512, 210, 120)
	for _, item := range []string{"Codex + Claude", "只写 Codex", "只写 Claude"} {
		comboAdd(target, item)
	}
	comboSelect(target, 0)
	create(idAdd, "BUTTON", "添加项目", bsPushButton|bsFlat|wsTabStop, 760, 502, 116, 40)
	list := create(idList, "LISTBOX", "", lbsNotify|lbsOwnerDrawFixed|lbsHasStrings|lbsNoIntegralHeight|wsVScroll|wsTabStop, 520, 588, 500, 240)
	pSendMessage.Call(list, lbSetItemHeight, 0, 52)
	pShowScrollBar.Call(list, sbVert, 1)
}

func (a *guiApp) paint() {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer pEndPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))

	client := clientRect(a.hwnd)
	pSetViewportOrgEx.Call(hdc, 0, signedIntPtr(-a.scrollY), 0)
	bgBottom := a.contentH
	if vis := client.bottom + a.scrollY; vis > bgBottom {
		bgBottom = vis
	}
	fillRect(hdc, rect{0, 0, client.right, bgBottom}, rgb(246, 248, 251))
	fillRect(hdc, rect{0, 0, client.right, 88}, rgb(255, 255, 255))

	drawRoundRect(hdc, rect{42, 25, 76, 59}, rgb(255, 245, 236), rgb(255, 203, 158), 14)
	drawTextRect(hdc, a.cardFont, "AF", rect{48, 30, 72, 56}, rgb(232, 99, 24), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(hdc, a.titleFont, "飞书提醒agent", rect{92, 20, 330, 52}, rgb(16, 83, 183), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(hdc, a.smallFont, "Codex / Claude Code 审批与完成提醒", rect{94, 52, 430, 76}, rgb(93, 104, 120), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)

	drawPill(hdc, a.smallFont, "飞书自建应用", rect{500, 28, 620, 60}, false)
	drawPill(hdc, a.smallFont, "Codex", rect{634, 28, 700, 60}, true)
	drawPill(hdc, a.smallFont, "Claude", rect{712, 28, 792, 60}, false)

	drawRoundRect(hdc, rect{40, 112, 1088, 368}, rgb(255, 255, 255), rgb(221, 226, 233), 18)
	drawStepBadge(hdc, a.cardFont, "1", rect{74, 146, 116, 188}, rgb(237, 253, 248), rgb(20, 158, 122))
	drawTextRect(hdc, a.cardFont, "连接飞书", rect{138, 132, 320, 162}, rgb(17, 24, 39), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(hdc, a.font, "生成二维码，用手机飞书扫码确认。", rect{138, 164, 470, 194}, rgb(13, 105, 235), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(hdc, a.smallFont, "应用名固定，凭证只保存在本机。", rect{138, 196, 430, 222}, rgb(89, 99, 115), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawPill(hdc, a.smallFont, "本地推送模式", rect{138, 232, 252, 260}, false)
	drawPill(hdc, a.smallFont, "无需公网", rect{264, 232, 352, 260}, false)

	drawRoundRect(hdc, rect{560, 132, 756, 328}, rgb(249, 250, 252), rgb(225, 230, 238), 14)
	a.drawQRCode(hdc, rect{572, 144, 744, 316})
	drawTextRect(hdc, a.smallFont, "状态", rect{780, 134, 870, 158}, rgb(89, 99, 115), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	statusText, statusColor := a.connectionStatus()
	drawTextRect(hdc, a.smallFont, statusText, rect{780, 158, 980, 182}, statusColor, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
	drawTextRect(hdc, a.smallFont, "接收人", rect{780, 242, 940, 266}, rgb(89, 99, 115), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawRoundRect(hdc, rect{780, 268, 1028, 304}, rgb(249, 250, 252), rgb(225, 230, 238), 12)

	drawRoundRect(hdc, rect{40, 392, 1088, 850}, rgb(255, 255, 255), rgb(221, 226, 233), 18)
	drawStepBadge(hdc, a.cardFont, "2", rect{74, 426, 116, 468}, rgb(238, 244, 255), rgb(52, 103, 246))
	drawTextRect(hdc, a.cardFont, "添加项目", rect{138, 408, 300, 438}, rgb(17, 24, 39), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(hdc, a.font, "选择项目文件夹，写入规则文件。", rect{138, 440, 390, 470}, rgb(13, 105, 235), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(hdc, a.smallFont, "重复添加会更新旧规则。", rect{138, 470, 360, 496}, rgb(89, 99, 115), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(hdc, a.smallFont, "项目文件夹", rect{520, 418, 660, 442}, rgb(89, 99, 115), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawRoundRect(hdc, rect{520, 442, 920, 480}, rgb(249, 250, 252), rgb(225, 230, 238), 12)
	drawRoundRect(hdc, rect{930, 438, 1016, 478}, rgb(247, 248, 250), rgb(216, 222, 231), 16)
	drawTextRect(hdc, a.smallFont, "写入范围", rect{520, 482, 660, 508}, rgb(89, 99, 115), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawRoundRect(hdc, rect{760, 502, 876, 542}, rgb(13, 105, 235), rgb(13, 105, 235), 18)
	drawTextRect(hdc, a.sectionFont, "已接入", rect{520, 556, 620, 582}, rgb(17, 24, 39), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(hdc, a.smallFont, "已写入规则的项目。", rect{600, 558, 820, 582}, rgb(89, 99, 115), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)

	drawRoundRect(hdc, rect{40, 868, 1088, 900}, rgb(238, 246, 255), rgb(188, 215, 254), 16)
	drawTextRect(hdc, a.smallFont, a.statusText, rect{58, 870, 1070, 898}, rgb(28, 76, 190), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
}

func (a *guiApp) drawProjectItem(item *drawItemStruct) {
	if item.itemID == 0xffffffff {
		return
	}
	selected := item.itemState&odsSelected != 0
	fill := rgb(255, 255, 255)
	stroke := rgb(222, 226, 232)
	textColor := rgb(14, 18, 27)
	pathColor := rgb(0, 103, 255)
	if selected {
		fill = rgb(232, 244, 255)
		stroke = rgb(75, 156, 255)
	}
	card := rect{
		left:   item.rcItem.left + 0,
		top:    item.rcItem.top + 5,
		right:  item.rcItem.right - 10,
		bottom: item.rcItem.bottom - 5,
	}
	drawRoundRect(item.hdc, card, fill, stroke, 16)

	text := listBoxText(item.hwndItem, item.itemID)
	title := filepath.Base(strings.Trim(text, `"`))
	if strings.TrimSpace(title) == "" || title == "." {
		title = "项目文件夹"
	}
	initial := firstInitial(title)
	icon := rect{card.left + 22, card.top + 14, card.left + 58, card.top + 50}
	drawRoundRect(item.hdc, icon, rgb(247, 248, 250), rgb(224, 227, 232), 14)
	drawTextRect(item.hdc, a.smallFont, initial, rect{icon.left + 11, icon.top + 4, icon.right - 7, icon.bottom - 4}, rgb(91, 101, 116), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	drawTextRect(item.hdc, a.font, title, rect{card.left + 76, card.top + 5, card.right - 18, card.top + 27}, textColor, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
	drawTextRect(item.hdc, a.smallFont, text, rect{card.left + 76, card.top + 27, card.right - 18, card.bottom - 4}, pathColor, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
}

func (a *guiApp) drawQRCode(hdc uintptr, rc rect) {
	a.mu.Lock()
	matrix := a.qrMatrix
	qrErr := a.qrErr
	registering := a.registering
	a.mu.Unlock()

	drawRoundRect(hdc, rc, rgb(255, 255, 255), rgb(216, 222, 231), 8)
	inner := rect{rc.left + 8, rc.top + 8, rc.right - 8, rc.bottom - 8}
	if qrErr != nil {
		drawTextRect(hdc, a.smallFont, "二维码生成失败", rect{inner.left, inner.top + 25, inner.right, inner.bottom}, rgb(185, 92, 23), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
		return
	}
	if len(matrix) == 0 {
		text := "点击生成"
		if registering {
			text = "生成中..."
		}
		drawTextRect(hdc, a.smallFont, text, rect{inner.left + 20, inner.top + 24, inner.right, inner.bottom}, rgb(89, 99, 115), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
		return
	}
	drawQRMatrix(hdc, matrix, inner)
}

func (a *guiApp) loadConfigToControls() {
	cfg, _ := loadConfig("")
	receiver := cfg.FeishuReceiveID
	if cfg.FeishuReceiveType != "" && receiver != "" {
		receiver = normalizeReceiveType(cfg.FeishuReceiveType) + ":" + receiver
	}
	setText(a.controls[idReceiver], receiver)
	setChecked(a.controls[idAutostart], isAutostartEnabled(a.installPath))
}

func (a *guiApp) refreshProjectList() {
	list := a.controls[idList]
	pSendMessage.Call(list, lbReset, 0, 0)
	cfg, _ := loadConfig("")
	for _, item := range cfg.ProjectFolders {
		pSendMessage.Call(list, lbAddString, 0, uintptr(unsafe.Pointer(utf16Ptr(item))))
	}
	pShowScrollBar.Call(list, sbVert, 1)
}

func (a *guiApp) connectionStatus() (string, uintptr) {
	a.mu.Lock()
	registering := a.registering
	hasQR := len(a.qrMatrix) > 0
	qrErr := a.qrErr
	a.mu.Unlock()

	if qrErr != nil {
		return "二维码生成失败", rgb(185, 92, 23)
	}
	if registering {
		if hasQR {
			return "等待手机扫码", rgb(13, 105, 235)
		}
		return "正在生成二维码", rgb(13, 105, 235)
	}
	cfg, err := loadConfig("")
	if err != nil || cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" || cfg.FeishuReceiveID == "" {
		if hasQR {
			return "等待扫码确认", rgb(13, 105, 235)
		}
		return "待扫码", rgb(185, 92, 23)
	}
	return "已连接", rgb(16, 132, 96)
}

func (a *guiApp) onCommand(id int) {
	switch id {
	case idConnect:
		a.registerApp()
	case idTest:
		a.sendTest()
	case idBrowse:
		if path := browseFolder(a.hwnd); path != "" {
			setText(a.controls[idProject], path)
		}
	case idAdd:
		a.addProject()
	case idHide:
		a.hideToTray()
	case idExit:
		a.removeTrayIcon()
		pDestroyWindow.Call(a.hwnd)
	}
}

func (a *guiApp) registerApp() {
	a.mu.Lock()
	if a.registering {
		a.mu.Unlock()
		a.setStatus("正在等待飞书扫码确认，请在手机飞书里完成授权。")
		return
	}
	a.registering = true
	a.lastAsyncErr = nil
	a.qrMatrix = nil
	a.qrURL = ""
	a.qrExpires = 0
	a.qrErr = nil
	a.mu.Unlock()

	a.setStatus("正在生成飞书二维码，请稍等...")
	go func() {
		err := runAppRegisterWithQRCode([]string{}, ioDiscardWriter{}, func(rawURL string, expires int) {
			matrix, qrErr := makeQRCodeMatrix(rawURL)
			a.mu.Lock()
			a.qrURL = rawURL
			a.qrExpires = expires
			a.qrMatrix = matrix
			a.qrErr = qrErr
			a.mu.Unlock()
			if qrErr != nil {
				guiLog("registerApp: QR matrix error: %v", qrErr)
			} else {
				guiLog("registerApp: QR ready url_len=%d expires=%d matrix=%d", len(rawURL), expires, len(matrix))
			}
			pPostMessage.Call(a.hwnd, wmAppQRCodeReady, 0, 0)
		})
		a.mu.Lock()
		a.lastAsyncErr = err
		a.registering = false
		a.mu.Unlock()
		pPostMessage.Call(a.hwnd, wmAppRegisterDone, 0, 0)
	}()
}

func (a *guiApp) finishRegisterApp() {
	a.mu.Lock()
	err := a.lastAsyncErr
	a.lastAsyncErr = nil
	a.mu.Unlock()
	if err != nil {
		a.setStatus("扫码连接失败：" + err.Error())
		messageBox(a.hwnd, err.Error(), "扫码连接失败")
		return
	}
	a.loadConfigToControls()
	if err := setAutostart(isChecked(a.controls[idAutostart]), a.installPath); err != nil {
		a.setStatus("开机自启设置失败：" + err.Error())
		messageBox(a.hwnd, err.Error(), "开机自启设置失败")
		return
	}
	a.setStatus("飞书自建应用已连接：现在可以添加项目文件夹。")
}

func (a *guiApp) addProject() {
	project := strings.TrimSpace(getText(a.controls[idProject]))
	if project == "" {
		a.setStatus("请选择一个项目文件夹。")
		return
	}
	target := "both"
	switch comboIndex(a.controls[idTarget]) {
	case 1:
		target = "codex"
	case 2:
		target = "claude"
	}
	results, err := addProjectRules(project, target, a.installPath, false)
	if err != nil {
		a.setStatus("添加失败：" + err.Error())
		messageBox(a.hwnd, err.Error(), "添加失败")
		return
	}
	cfg, _ := loadConfig("")
	cfg.DefaultAgent = "Codex"
	abs, _ := filepath.Abs(strings.Trim(project, `"`))
	cfg.ProjectFolders = appendUniquePath(cfg.ProjectFolders, abs)
	_ = writeConfig(configPath(""), cfg)
	a.refreshProjectList()
	if cfg.FeishuAppID == "" || cfg.FeishuReceiveID == "" {
		a.setStatus("项目规则已写入；还需要扫码连接飞书自建应用。")
		return
	}
	a.setStatus("项目规则已写入：" + strings.Join(results, " | "))
}

func (a *guiApp) sendTest() {
	cfg, err := loadConfig("")
	if err != nil || cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" || cfg.FeishuReceiveID == "" {
		a.setStatus("请先生成二维码并完成飞书扫码连接。")
		return
	}
	a.setStatus("正在发送飞书测试通知...")
	go func() {
		err := sendTestNotice(ioDiscardWriter{}, cfg, a.installDir, false)
		a.mu.Lock()
		a.lastAsyncErr = err
		a.mu.Unlock()
		pPostMessage.Call(a.hwnd, wmAppTestDone, 0, 0)
	}()
}

func (a *guiApp) setStatus(text string) {
	a.statusText = text
	pInvalidateRect.Call(a.hwnd, 0, 1)
}

func (a *guiApp) hideToTray() {
	pShowWindow.Call(a.hwnd, swHide)
}

func (a *guiApp) restore() {
	pShowWindow.Call(a.hwnd, swShow)
	pUpdateWindow.Call(a.hwnd)
}

func (a *guiApp) clientHeight() int32 {
	client := clientRect(a.hwnd)
	return client.bottom - client.top
}

func (a *guiApp) maxScroll() int32 {
	m := a.contentH - a.clientHeight()
	if m < 0 {
		return 0
	}
	return m
}

func (a *guiApp) updateScroll() {
	if a.hwnd == 0 {
		return
	}
	a.contentH = guiContentHeight
	if a.scrollY > a.maxScroll() {
		a.scrollY = a.maxScroll()
	}
	if a.scrollY < 0 {
		a.scrollY = 0
	}
	si := scrollInfo{
		cbSize: uint32(unsafe.Sizeof(scrollInfo{})),
		fMask:  sifRange | sifPage | sifPos,
		nMin:   0,
		nMax:   a.contentH - 1,
		nPage:  uint32(a.clientHeight()),
		nPos:   a.scrollY,
	}
	pSetScrollInfo.Call(a.hwnd, sbVert, uintptr(unsafe.Pointer(&si)), 1)
	a.applyScroll()
}

func (a *guiApp) applyScroll() {
	for id, p := range a.design {
		h := a.controls[id]
		if h == 0 {
			continue
		}
		pMoveWindow.Call(h, signedIntPtr(p.x), signedIntPtr(p.y-a.scrollY), uintptr(p.w), uintptr(p.h), 1)
	}
	pInvalidateRect.Call(a.hwnd, 0, 1)
}

func (a *guiApp) onVScroll(action int, _ int32) {
	pos := a.scrollY
	page := a.clientHeight()
	switch action {
	case sbLineUp:
		pos -= 48
	case sbLineDown:
		pos += 48
	case sbPageUp:
		pos -= page
	case sbPageDown:
		pos += page
	case sbThumbTrack, sbThumbPosition:
		var si scrollInfo
		si.cbSize = uint32(unsafe.Sizeof(si))
		si.fMask = sifTrackPos
		pGetScrollInfo.Call(a.hwnd, sbVert, uintptr(unsafe.Pointer(&si)))
		pos = si.nTrackPos
	case sbTop:
		pos = 0
	case sbBottom:
		pos = a.contentH
	}
	a.setScrollY(pos)
}

func (a *guiApp) onMouseWheel(delta int32) {
	a.setScrollY(a.scrollY - (delta/120)*60)
}

func (a *guiApp) setScrollY(pos int32) {
	if pos > a.maxScroll() {
		pos = a.maxScroll()
	}
	if pos < 0 {
		pos = 0
	}
	if pos == a.scrollY {
		return
	}
	a.scrollY = pos
	si := scrollInfo{
		cbSize: uint32(unsafe.Sizeof(scrollInfo{})),
		fMask:  sifPos,
		nPos:   a.scrollY,
	}
	pSetScrollInfo.Call(a.hwnd, sbVert, uintptr(unsafe.Pointer(&si)), 1)
	a.applyScroll()
}

func getWorkArea() rect {
	var rc rect
	ok, _, _ := pSystemParameters.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&rc)), 0)
	if ok == 0 || rc.right <= rc.left || rc.bottom <= rc.top {
		return rect{0, 0, 1160, 872}
	}
	return rc
}

func fitWindowSize(desiredW, desiredH int32) (int32, int32) {
	wa := getWorkArea()
	maxW := wa.right - wa.left
	maxH := wa.bottom - wa.top
	w, h := desiredW, desiredH
	if maxW > 0 && w > maxW {
		w = maxW
	}
	if maxH > 0 && h > maxH {
		h = maxH
	}
	return w, h
}

func (a *guiApp) addTrayIcon() {
	guiLog("addTrayIcon: start")
	icon, _, _ := pLoadIcon.Call(0, uintptr(32512))
	var data notifyIconData
	data.cbSize = uint32(unsafe.Sizeof(data))
	data.hWnd = a.hwnd
	data.uID = 1
	data.uFlags = nifMessage | nifIcon | nifTip
	data.uCallbackMessage = wmAppTray
	data.hIcon = icon
	copy(data.szTip[:], syscall.StringToUTF16("Agent Feishu"))
	ret, _, err := pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
	guiLog("addTrayIcon: Shell_NotifyIcon ret=%d err=%v", ret, err)
}

func (a *guiApp) removeTrayIcon() {
	var data notifyIconData
	data.cbSize = uint32(unsafe.Sizeof(data))
	data.hWnd = a.hwnd
	data.uID = 1
	pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmEraseBkgnd:
		return 1
	case wmSize:
		if activeGUI != nil {
			activeGUI.updateScroll()
		}
		return 0
	case wmVScroll:
		if activeGUI != nil {
			activeGUI.onVScroll(int(wParam&0xffff), int32(int16((wParam>>16)&0xffff)))
		}
		return 0
	case wmMouseWheel:
		if activeGUI != nil {
			activeGUI.onMouseWheel(int32(int16((wParam >> 16) & 0xffff)))
		}
		return 0
	case wmPaint:
		if activeGUI != nil {
			activeGUI.paint()
			return 0
		}
	case wmMeasureItem:
		measure := (*measureItemStruct)(unsafe.Pointer(lParam))
		if measure != nil && measure.ctlID == idList {
			measure.itemHeight = 52
			return 1
		}
	case wmDrawItem:
		item := (*drawItemStruct)(unsafe.Pointer(lParam))
		if item != nil && item.ctlID == idList && activeGUI != nil {
			activeGUI.drawProjectItem(item)
			return 1
		}
	case wmCommand:
		if activeGUI != nil {
			activeGUI.onCommand(int(wParam & 0xffff))
		}
		return 0
	case wmClose:
		if activeGUI != nil {
			activeGUI.hideToTray()
		}
		return 0
	case wmAppTray:
		if lParam == wmLButtonD && activeGUI != nil {
			activeGUI.restore()
		}
		return 0
	case wmAppRegisterDone:
		if activeGUI != nil {
			activeGUI.finishRegisterApp()
			return 0
		}
	case wmAppTestDone:
		if activeGUI != nil {
			activeGUI.finishTest()
			return 0
		}
	case wmAppQRCodeReady:
		if activeGUI != nil {
			activeGUI.mu.Lock()
			qrErr := activeGUI.qrErr
			activeGUI.mu.Unlock()
			if qrErr != nil {
				activeGUI.setStatus("二维码生成失败：" + qrErr.Error())
				return 0
			}
			activeGUI.setStatus("二维码已生成：请用手机飞书扫码，并确认创建「飞书提醒agent」。")
			return 0
		}
	case wmDestroy:
		if activeGUI != nil {
			activeGUI.removeTrayIcon()
		}
		pPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := pDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func (a *guiApp) finishTest() {
	a.mu.Lock()
	err := a.lastAsyncErr
	a.lastAsyncErr = nil
	a.mu.Unlock()
	if err != nil {
		a.setStatus("测试通知发送失败：" + err.Error())
		messageBox(a.hwnd, err.Error(), "测试通知发送失败")
		return
	}
	a.setStatus("测试通知已发送，请查看飞书。")
}

func getText(hwnd uintptr) string {
	n, _, _ := pGetWindowTextLength.Call(hwnd)
	buf := make([]uint16, int(n)+1)
	pGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func clientRect(hwnd uintptr) rect {
	rc := rect{0, 0, 1180, 845}
	if hwnd == 0 {
		return rc
	}
	ret, _, _ := pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	if ret == 0 || rc.right <= rc.left || rc.bottom <= rc.top {
		return rect{0, 0, 1180, 845}
	}
	return rc
}

func rgb(r, g, b byte) uintptr {
	return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16)
}

func signedIntPtr(v int32) uintptr {
	return uintptr(int(v))
}

func fillRect(hdc uintptr, rc rect, color uintptr) {
	brush, _, _ := pCreateSolidBrush.Call(color)
	if brush == 0 {
		return
	}
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), brush)
	pDeleteObject.Call(brush)
}

func fillSolidRect(hdc uintptr, rc rect, color uintptr) {
	brush, _, _ := pCreateSolidBrush.Call(color)
	pen, _, _ := pCreatePen.Call(psSolid, 1, color)
	if brush == 0 || pen == 0 {
		if brush != 0 {
			pDeleteObject.Call(brush)
		}
		if pen != 0 {
			pDeleteObject.Call(pen)
		}
		return
	}
	oldBrush, _, _ := pSelectObject.Call(hdc, brush)
	oldPen, _, _ := pSelectObject.Call(hdc, pen)
	pRectangle.Call(hdc, uintptr(rc.left), uintptr(rc.top), uintptr(rc.right), uintptr(rc.bottom))
	if oldBrush != 0 {
		pSelectObject.Call(hdc, oldBrush)
	}
	if oldPen != 0 {
		pSelectObject.Call(hdc, oldPen)
	}
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)
}

func drawQRMatrix(hdc uintptr, matrix [][]bool, rc rect) {
	size := int32(len(matrix))
	if size == 0 {
		return
	}
	width := rc.right - rc.left
	height := rc.bottom - rc.top
	cell := width / size
	if hCell := height / size; hCell < cell {
		cell = hCell
	}
	if cell <= 0 {
		cell = 1
	}
	total := cell * size
	left := rc.left + (width-total)/2
	top := rc.top + (height-total)/2
	black := rgb(17, 24, 39)
	for y := int32(0); y < size; y++ {
		for x := int32(0); x < size; x++ {
			if matrix[y][x] {
				fillSolidRect(hdc, rect{
					left:   left + x*cell,
					top:    top + y*cell,
					right:  left + (x+1)*cell,
					bottom: top + (y+1)*cell,
				}, black)
			}
		}
	}
}

func drawStepBadge(hdc uintptr, font uintptr, text string, rc rect, fill uintptr, color uintptr) {
	drawRoundRect(hdc, rc, fill, rgb(222, 230, 240), 18)
	drawTextRect(hdc, font, text, rect{rc.left + 14, rc.top + 4, rc.right - 10, rc.bottom - 4}, color, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
}

func drawRoundRect(hdc uintptr, rc rect, fill uintptr, stroke uintptr, radius int32) {
	brush, _, _ := pCreateSolidBrush.Call(fill)
	pen, _, _ := pCreatePen.Call(psSolid, 1, stroke)
	if brush == 0 || pen == 0 {
		if brush != 0 {
			pDeleteObject.Call(brush)
		}
		if pen != 0 {
			pDeleteObject.Call(pen)
		}
		return
	}
	oldBrush, _, _ := pSelectObject.Call(hdc, brush)
	oldPen, _, _ := pSelectObject.Call(hdc, pen)
	pRoundRect.Call(hdc, uintptr(rc.left), uintptr(rc.top), uintptr(rc.right), uintptr(rc.bottom), uintptr(radius), uintptr(radius))
	if oldBrush != 0 {
		pSelectObject.Call(hdc, oldBrush)
	}
	if oldPen != 0 {
		pSelectObject.Call(hdc, oldPen)
	}
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)
}

func drawTextRect(hdc uintptr, font uintptr, text string, rc rect, color uintptr, flags uint32) {
	if strings.TrimSpace(text) == "" {
		return
	}
	if font != 0 {
		old, _, _ := pSelectObject.Call(hdc, font)
		defer func() {
			if old != 0 {
				pSelectObject.Call(hdc, old)
			}
		}()
	}
	pSetBkMode.Call(hdc, transparent)
	pSetTextColor.Call(hdc, color)
	utf := syscall.StringToUTF16(text)
	pDrawText.Call(
		hdc,
		uintptr(unsafe.Pointer(&utf[0])),
		uintptr(len(utf)-1),
		uintptr(unsafe.Pointer(&rc)),
		uintptr(flags),
	)
}

func drawPill(hdc uintptr, font uintptr, text string, rc rect, selected bool) {
	fill := rgb(246, 247, 249)
	stroke := rgb(238, 240, 244)
	color := rgb(91, 101, 116)
	if selected {
		fill = rgb(238, 246, 255)
		stroke = rgb(191, 219, 254)
		color = rgb(10, 94, 207)
	}
	drawRoundRect(hdc, rc, fill, stroke, 18)
	drawTextRect(hdc, font, text, rect{rc.left + 14, rc.top + 2, rc.right - 12, rc.bottom - 2}, color, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
}

func listBoxText(hwnd uintptr, index uint32) string {
	length, _, _ := pSendMessage.Call(hwnd, lbGetTextLen, uintptr(index), 0)
	if int(length) < 0 || length > 4096 {
		return ""
	}
	buf := make([]uint16, int(length)+1)
	pSendMessage.Call(hwnd, lbGetText, uintptr(index), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}

func firstInitial(text string) string {
	for _, r := range strings.TrimSpace(text) {
		return strings.ToUpper(string(r))
	}
	return "P"
}

func setText(hwnd uintptr, value string) {
	pSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(value))))
}

func comboAdd(hwnd uintptr, value string) {
	pSendMessage.Call(hwnd, cbAddString, 0, uintptr(unsafe.Pointer(utf16Ptr(value))))
}

func comboSelect(hwnd uintptr, idx int) {
	pSendMessage.Call(hwnd, cbSetCurSel, uintptr(idx), 0)
}

func comboIndex(hwnd uintptr) int {
	r, _, _ := pSendMessage.Call(hwnd, cbGetCurSel, 0, 0)
	return int(r)
}

func setChecked(hwnd uintptr, checked bool) {
	value := uintptr(0)
	if checked {
		value = bstChecked
	}
	pSendMessage.Call(hwnd, bmSetCheck, value, 0)
}

func isChecked(hwnd uintptr) bool {
	r, _, _ := pSendMessage.Call(hwnd, bmGetCheck, 0, 0)
	return r == bstChecked
}

func browseFolder(owner uintptr) string {
	guiLog("browseFolder: start")
	var display [260]uint16
	title := utf16Ptr("选择要接入 Agent Feishu 的项目文件夹")
	info := browseInfo{
		hwndOwner:      owner,
		pszDisplayName: &display[0],
		lpszTitle:      title,
		ulFlags:        0x0001 | 0x0040,
	}
	pidl, _, _ := pSHBrowseForFolder.Call(uintptr(unsafe.Pointer(&info)))
	if pidl == 0 {
		guiLog("browseFolder: cancelled or failed")
		return ""
	}
	defer pCoTaskMemFree.Call(pidl)
	var path [260]uint16
	ok, _, _ := pSHGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		guiLog("browseFolder: SHGetPathFromIDList failed")
		return ""
	}
	selected := syscall.UTF16ToString(path[:])
	guiLog("browseFolder: selected=%s", selected)
	return selected
}

func messageBox(hwnd uintptr, text, title string) {
	pMessageBox.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(title))), 0)
}

func isAutostartEnabled(exePath string) bool {
	key, err := openRunKey(keyQueryValue)
	if err != nil {
		return false
	}
	defer pRegCloseKey.Call(key)

	value, err := readRegistryString(key, startupValueName)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(exePath))
}

func setAutostart(enabled bool, exePath string) error {
	if enabled {
		key, err := createRunKey()
		if err != nil {
			return err
		}
		defer pRegCloseKey.Call(key)
		return writeRegistryString(key, startupValueName, fmt.Sprintf("%q ui --start-hidden", exePath))
	}

	key, err := openRunKey(keySetValue)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok && errno == errFileNotFound {
			return nil
		}
		return err
	}
	defer pRegCloseKey.Call(key)
	return deleteRegistryValue(key, startupValueName)
}

func createRunKey() (uintptr, error) {
	var key uintptr
	ret, _, _ := pRegCreateKeyEx.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(utf16Ptr(startupRunKey))),
		0,
		0,
		0,
		keySetValue,
		0,
		uintptr(unsafe.Pointer(&key)),
		0,
	)
	if ret != 0 {
		return 0, syscall.Errno(ret)
	}
	return key, nil
}

func openRunKey(access uintptr) (uintptr, error) {
	var key uintptr
	ret, _, _ := pRegOpenKeyEx.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(utf16Ptr(startupRunKey))),
		0,
		access,
		uintptr(unsafe.Pointer(&key)),
	)
	if ret != 0 {
		return 0, syscall.Errno(ret)
	}
	return key, nil
}

func writeRegistryString(key uintptr, name, value string) error {
	data := syscall.StringToUTF16(value)
	ret, _, _ := pRegSetValueEx.Call(
		key,
		uintptr(unsafe.Pointer(utf16Ptr(name))),
		0,
		regSz,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)*2),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

func readRegistryString(key uintptr, name string) (string, error) {
	var dataType uint32
	var size uint32
	ret, _, _ := pRegQueryValueEx.Call(
		key,
		uintptr(unsafe.Pointer(utf16Ptr(name))),
		0,
		uintptr(unsafe.Pointer(&dataType)),
		0,
		uintptr(unsafe.Pointer(&size)),
	)
	if ret != 0 {
		return "", syscall.Errno(ret)
	}
	if dataType != regSz || size == 0 {
		return "", nil
	}
	buf := make([]uint16, int(size)/2)
	ret, _, _ = pRegQueryValueEx.Call(
		key,
		uintptr(unsafe.Pointer(utf16Ptr(name))),
		0,
		uintptr(unsafe.Pointer(&dataType)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret != 0 {
		return "", syscall.Errno(ret)
	}
	return syscall.UTF16ToString(buf), nil
}

func deleteRegistryValue(key uintptr, name string) error {
	ret, _, _ := pRegDeleteValue.Call(key, uintptr(unsafe.Pointer(utf16Ptr(name))))
	if ret != 0 && ret != errFileNotFound {
		return syscall.Errno(ret)
	}
	return nil
}

func createFont(height int32, weight int32) uintptr {
	font, _, _ := pCreateFont.Call(
		uintptr(uint32(-height)),
		0,
		0,
		0,
		uintptr(weight),
		0,
		0,
		0,
		1,
		0,
		0,
		5,
		0,
		uintptr(unsafe.Pointer(utf16Ptr("Segoe UI"))),
	)
	return font
}

func guiLog(format string, args ...any) {
	dir := defaultInstallDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	line := time.Now().Format(time.RFC3339Nano) + " " + fmt.Sprintf(format, args...) + "\r\n"
	file := filepath.Join(dir, "agent-feishu.log")
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func initializeCOM() bool {
	ret, _, _ := pCoInitializeEx.Call(0, coinitApartmentThreaded)
	if ret == 0 || ret == 1 {
		guiLog("initializeCOM: ok ret=%d", ret)
		return true
	}
	guiLog("initializeCOM: continuing without COM ret=%d", ret)
	return false
}

func utf16Ptr(value string) *uint16 {
	ptr, _ := syscall.UTF16PtrFromString(value)
	return ptr
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
