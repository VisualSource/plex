package devtools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
)

func StartRemoteDebuggingServer(logger *slog.Logger, ctx context.Context, port int) error {
	config := net.ListenConfig{}
	listener, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	logger.DebugContext(ctx, "starting remote devtools", slog.Int("port", port))

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			logger.ErrorContext(ctx, "failed to accept connection", slog.String("error", err.Error()))
			continue
		}

		go handleConnection(logger, ctx, conn)
	}

	return nil
}

func handleConnection(logger *slog.Logger, ctx context.Context, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	err := writePacket(logger, writer, map[string]any{
		"from":            "root",
		"applicationType": "browser",
		"traits": map[string]bool{
			"networkMonitor": false,
		},
	})
	if err != nil {
		logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
		return
	}

	logger.DebugContext(ctx, "new connection", slog.String("remote", conn.RemoteAddr().String()))

	for {
		message, err := readPacket(reader)
		if err != nil {
			logger.ErrorContext(ctx, "packet read error", slog.String("error", err.Error()))
			return
		}

		logger.DebugContext(ctx, "read", slog.Any("packet", message))

		switch message["to"].(string) {
		case "root":
			switch message["type"].(string) {
			case "connect":
				if err := writePacket(logger, writer, map[string]any{
					"from": "root",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getRoot":
				//https://searchfox.org/firefox-main/source/devtools/shared/specs/root.js
				if err := writePacket(logger, writer, map[string]any{
					"from":            "root",
					"preferenceActor": "plex.conn0.preference",
					"deviceActor":     "plex.conn0.device1",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "listTabs":
				if err := writePacket(logger, writer, map[string]any{
					"from":     "root",
					"selected": 0,
					"tabs": []map[string]any{
						{
							"actor":             "plex.conn0.tabDescriptor1",
							"browserId":         1,
							"browsingContextID": nil,
							"outerWindowID":     1,
							"selected":          true,
							"traits": map[string]bool{
								"watcher":                  true,
								"supportsReloadDescriptor": false,
								"supportsNavigation":       false,
							},
							"title": "Plex",
							"url":   "about:blank",
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "listAddons":
				if err := writePacket(logger, writer, map[string]any{
					"from":   "root",
					"addons": []map[string]any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getTab":
				// borwserId is also herer
				if err := writePacket(logger, writer, map[string]any{
					"from": "root",
					"tab": map[string]any{
						"actor":             "plex.conn0.tabDescriptor1",
						"browserId":         message["browserId"].(float64),
						"browsingContextID": nil,
						"isZombieTab":       false,
						"outerWindowID":     1,
						"selected":          true,
						"title":             "Plex",
						"traits": map[string]bool{
							"watcher":                  true, // needed for firefox 102+
							"supportsReloadDescriptor": false,
							"supportsNavigation":       false,
						},
						"url": "about:blank",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "listProcesses":
				if err := writePacket(logger, writer, map[string]any{
					"from":      message["to"],
					"processes": []Payload{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getProcess":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"processDescriptor": Payload{
						"actor":              "plex.conn0.processDescriptor1",
						"id":                 0,
						"isParent":           true,
						"isWindowlessParent": false,
						"traits": Payload{
							"watcher":                  true,
							"supportsReloadDescriptor": true,
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "listWorkers":
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"workers": []Payload{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "listServiceWorkerRegistrations":
				if err := writePacket(logger, writer, map[string]any{
					"from":          message["to"],
					"registrations": []Payload{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.device1":
			switch message["type"].(string) {
			case "getDescription":
				if err := writePacket(logger, writer, map[string]any{
					"from": "plex.conn0.device1",
					"value": map[string]any{
						"apptype":   "browser",
						"name":      "Plex",
						"vender":    "Plex",
						"brandName": "Plex",
						"version":   "0.1.0",
						"channel":   "release",
						"os":        "Linux",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.preference":
			switch message["type"].(string) {
			case "getBoolPref":
				if err := writePacket(logger, writer, map[string]any{
					"from":  "plex.conn0.preference",
					"value": false,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.tabDescriptor1":
			switch message["type"].(string) {
			case "getWatcher":
				err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"actor": "plex.conn0.watcher1",
					"traits": map[string]any{
						"frame":          true,
						"process":        false,
						"worker":         false,
						"service_worker": false,
						"shared_worker":  false,
						"content_script": false,
						"resources": map[string]bool{
							"console-message": false,
							"document-event":  false,
							"error-message":   false,
							"source":          true,
						},
					},
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getFavicon":
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"favicon": "",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.watcher1":
			switch message["type"].(string) {
			case "watchTargets":
				targetType := message["targetType"].(string)

				if targetType == "frame" {
					err := writePacket(logger, writer, map[string]any{
						"from": message["to"],
						"type": "target-available-form",
						"target": map[string]any{
							"actor":                       "plex.conn0.windowGlobalTarget1",
							"targetType":                  "frame",
							"browsingContextID":           1,
							"processID":                   1,
							"followWindowGlobalLifeCycle": true,
							"innerWindowId":               1,
							"parentInnerWindowId":         0,
							"topInnerWindowId":            1,
							"isTopLevelTarget":            true,
							"ignoreSubFrames":             false,
							"isPopup":                     false,
							"isPrivate":                   false,
							"title":                       "Plex",
							"url":                         "about:blank",
							"outerWindowID":               1,
							"isFallbackExtensionDocument": false,
							"addonId":                     nil,

							"traits": map[string]bool{
								"isBrowsingContext":          true,
								"supportsTopLevelTargetFlag": true,
								"frames":                     true,
								"logInPage":                  true,
								"watchpoints":                true,
								"navigation":                 true,
							},

							// Sub-actor IDs — only advertise what you implement.
							// Each one becomes a new "to" target the client will start calling.
							"consoleActor":       "plex.conn0.console1",
							"threadActor":        "plex.conn0.thread1",
							"inspectorActor":     "plex.conn0.inspector1",
							"cssPropertiesActor": "plex.conn0.cssProperties1",
							"accessibilityActor": "plex.conn0.accessibility1",
							"reflowActor":        "plex.conn0.reflow1",
						},
					})
					if err != nil {
						logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
						return
					}
				}

				if err := writePacket(logger, writer, map[string]any{
					"from": "plex.conn0.watcher1",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			case "getThreadConfigurationActor":

				if err := writePacket(logger, writer, map[string]any{
					"from": "plex.conn0.watcher1",
					"configuration": map[string]any{
						"actor": "plex.conn0.threadConfiguration1",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			case "getTargetConfigurationActor":
				if err := writePacket(logger, writer, map[string]any{
					"from": "plex.conn0.watcher1",
					"configuration": map[string]any{
						"actor":         "plex.conn0.targetConfiguration1",
						"configuration": map[string]any{},
						"traits": map[string]any{
							"supportedOptions": map[string]bool{},
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			case "getParentBrowsingContextID":
				if err := writePacket(logger, writer, map[string]any{
					"from":              "plex.conn0.watcher1",
					"browsingContextID": nil,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getNetworkParentActor":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"nextworkParent": map[string]any{
						"actor": "plex.conn0.networkParent1",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "unwatchTargets":
				// no response
			default:
				logger.DebugContext(ctx, "unhandled packet")
				err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.targetConfiguration1", "plex.conn0.threadConfiguration1":
			switch message["type"] {
			case "updateConfiguration":
				err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}

		case "plex.conn0.inspector1":
			switch message["type"].(string) {
			case "getWalker":
				err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"walker": map[string]any{
						"actor":             "plex.conn0.walker1",
						"rfpCSSColorScheme": false,
						"traits":            map[string]any{},
						"root": map[string]any{
							"actor":                   "plex.conn0.node1",
							"baseURI":                 "about:blank",
							"parent":                  nil,
							"nodeType":                9, // DOCUMENT_NODE
							"namespaceURI":            "http://www.w3.org/1999/xhtml",
							"nodeName":                "#document",
							"nodeValue":               nil,
							"displayName":             "#document",
							"numChildren":             1, // 1 = the <html> element
							"displayType":             nil,
							"isScrollable":            false,
							"isTopLevelDocument":      true,
							"causesOverflow":          false,
							"containerType":           nil,
							"anchorName":              nil,
							"attrs":                   []any{},
							"customElementLocation":   nil,
							"isPseudoElement":         false,
							"isNativeAnonymous":       false,
							"isShadowRoot":            false,
							"shadowRootMode":          nil,
							"isShadowHost":            false,
							"isDirectShadowHostChild": false,
							"pseudoClassLocks":        []any{},
							"mutationBreakpoints":     map[string]any{},
							"isDisplayed":             true,
							"isInHTMLDocument":        true,
							"traits":                  map[string]any{},
							"browsingContextID":       1,
						},
					},
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
				// Push root-available immediately while no walker requests are pending.
				// Sending it later (from watchRootNode) races with getLayoutInspector.
				if err := writePacket(logger, writer, map[string]any{
					"from": "plex.conn0.walker1",
					"type": "root-available",
					"node": map[string]any{
						"actor":                   "plex.conn0.node1",
						"baseURI":                 "about:blank",
						"parent":                  nil,
						"nodeType":                9,
						"namespaceURI":            "http://www.w3.org/1999/xhtml",
						"nodeName":                "#document",
						"nodeValue":               nil,
						"displayName":             "#document",
						"numChildren":             1,
						"displayType":             nil,
						"isScrollable":            false,
						"isTopLevelDocument":      true,
						"causesOverflow":          false,
						"containerType":           nil,
						"anchorName":              nil,
						"attrs":                   []any{},
						"customElementLocation":   nil,
						"isPseudoElement":         false,
						"isNativeAnonymous":       false,
						"isShadowRoot":            false,
						"shadowRootMode":          nil,
						"isShadowHost":            false,
						"isDirectShadowHostChild": false,
						"pseudoClassLocks":        []any{},
						"mutationBreakpoints":     map[string]any{},
						"isDisplayed":             true,
						"isInHTMLDocument":        true,
						"traits":                  map[string]any{},
						"browsingContextID":       1,
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getPageStyle":
				err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"pageStyle": map[string]any{
						"actor": "plex.conn0.pageStyle1",
						"traits": map[string]bool{
							"fontStretchLevel4": false,
							"fontStyleLevel4":   false,
							"fontVariations":    false,
							"fontWeightLevel4":  false,
						},
					},
				})
				if err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getHighlighterByType":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"highlighter": map[string]any{
						"actor": "plex.conn0.customHighlighter1",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "supportsHighlighters":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"value": true,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, Payload{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.windowGlobalTarget1":
			switch message["type"].(string) {
			case "listFrames":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"frames": []map[string]any{
						{
							"id":         1,
							"parentID":   nil,
							"isTopLevel": true,
							"url":        "about:blank",
							"title":      "Plex",
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "detach":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "attach":
				if err := writePacket(logger, writer, map[string]any{
					"from":        message["to"],
					"type":        "tabAttached",
					"threadActor": "plex.conn0.thread1",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.customHighlighter1":
			switch message["type"].(string) {
			case "show":
				if err := writePacket(logger, writer, Payload{
					"from":  message["to"],
					"value": true,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "finalize":
				// NO response
			case "hide":
				if err := writePacket(logger, writer, Payload{
					"from": message["to"],
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.node2":
			switch message["type"].(string) {
			case "getUniqueSelector":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"value": "body",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			}

		case "plex.conn0.nodeHtml1":
			switch message["type"].(string) {
			case "getLayoutInspector":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"actor": map[string]any{
						"actor": "plex.conn0.layout1",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getUniqueSelector":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"value": "#document",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.node1":
			switch message["type"].(string) {
			case "getLayoutInspector":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"actor": map[string]any{
						"actor": "plex.conn0.layout1",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getUniqueSelector":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"value": "#document",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}

		case "plex.conn0.walker1":
			switch message["type"].(string) {
			case "querySelector": // node, selector

				if err := writePacket(logger, writer, Payload{
					"from": message["to"],
					"node": Payload{
						"actor":               "plex.conn0.node2",
						"nodeType":            1,
						"namespaceURI":        "http://www.w3.org/1999/xhtml",
						"nodeName":            "BODY",
						"nodeValue":           nil,
						"displayName":         "body",
						"numChildren":         0,
						"attrs":               []any{},
						"isInHTMLDocument":    true,
						"pseudoClassLocks":    []any{},
						"mutationBreakpoints": map[string]any{},
						"traits":              map[string]any{},
						"browsingContextID":   1,
						"isDisplayed":         true,
						"parent":              "plex.conn0.nodeHtml1",
					},
					"newParents": []Payload{
						{
							"actor":               "plex.conn0.nodeHtml1",
							"nodeType":            1,
							"namespaceURI":        "http://www.w3.org/1999/xhtml",
							"nodeName":            "HTML",
							"nodeValue":           nil,
							"displayName":         "html",
							"numChildren":         2,
							"attrs":               []any{},
							"isInHTMLDocument":    true,
							"pseudoClassLocks":    []any{},
							"mutationBreakpoints": map[string]any{},
							"traits":              map[string]any{},
							"browsingContextID":   1,
							"isDisplayed":         true,
							"parent":              "plex.conn0.node1",
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			case "watchRootNode":
				// root-available was already pushed after getWalker; no response needed here.
			case "getUniqueSelector":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"value": "body", // or whatever selector fits the node
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getLayoutInspector":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"actor": "plex.conn0.layout1",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "children":
				nodeActor := message["node"].(string)

				var nodes []any
				switch nodeActor {
				case "plex.conn0.node1": // #document → return <html>
					nodes = []any{
						map[string]any{
							"actor":               "plex.conn0.nodeHtml1",
							"nodeType":            1,
							"namespaceURI":        "http://www.w3.org/1999/xhtml",
							"nodeName":            "HTML",
							"nodeValue":           nil,
							"displayName":         "html",
							"numChildren":         2,
							"attrs":               []any{},
							"isInHTMLDocument":    true,
							"pseudoClassLocks":    []any{},
							"mutationBreakpoints": map[string]any{},
							"traits":              map[string]any{},
							"browsingContextID":   1,
							"isDisplayed":         true,
							"parent":              "plex.conn0.node1",
						},
					}
				case "plex.conn0.nodeHtml1":
					nodes = []any{
						map[string]any{
							"actor":               "plex.conn0.nodeHead1",
							"nodeType":            1,
							"namespaceURI":        "http://www.w3.org/1999/xhtml",
							"nodeName":            "HEAD",
							"nodeValue":           nil,
							"displayName":         "head",
							"numChildren":         0,
							"attrs":               []any{},
							"isInHTMLDocument":    true,
							"pseudoClassLocks":    []any{},
							"mutationBreakpoints": map[string]any{},
							"traits":              map[string]any{},
							"browsingContextID":   1,
							"isDisplayed":         true,
							"parent":              "plex.conn0.nodeHtml1",
						},
						map[string]any{
							"actor":               "plex.conn0.node2", // must match querySelector
							"nodeType":            1,
							"namespaceURI":        "http://www.w3.org/1999/xhtml",
							"nodeName":            "BODY",
							"nodeValue":           nil,
							"displayName":         "body",
							"numChildren":         0,
							"attrs":               []any{},
							"isInHTMLDocument":    true,
							"pseudoClassLocks":    []any{},
							"mutationBreakpoints": map[string]any{},
							"traits":              map[string]any{},
							"browsingContextID":   1,
							"isDisplayed":         true,
							"parent":              "plex.conn0.nodeHtml1",
						},
					}
				default:
					nodes = []any{}
				}

				if err := writePacket(logger, writer, map[string]any{
					"from":     message["to"],
					"nodes":    nodes,
					"hasFirst": true,
					"hasLast":  true,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getOffsetParent":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"node": nil,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getAncestors":
				nodeActor := message["node"].(string)
				htmlForm := map[string]any{
					"actor":               "plex.conn0.nodeHtml1",
					"nodeType":            1,
					"namespaceURI":        "http://www.w3.org/1999/xhtml",
					"nodeName":            "HTML",
					"nodeValue":           nil,
					"displayName":         "html",
					"numChildren":         2,
					"attrs":               []any{},
					"isInHTMLDocument":    true,
					"pseudoClassLocks":    []any{},
					"mutationBreakpoints": map[string]any{},
					"traits":              map[string]any{},
					"browsingContextID":   1,
					"isDisplayed":         true,
					"parent":              "plex.conn0.node1",
				}
				docForm := map[string]any{
					"actor":                   "plex.conn0.node1",
					"nodeType":                9,
					"namespaceURI":            "http://www.w3.org/1999/xhtml",
					"nodeName":                "#document",
					"nodeValue":               nil,
					"displayName":             "#document",
					"numChildren":             1,
					"attrs":                   []any{},
					"isInHTMLDocument":        true,
					"isTopLevelDocument":      true,
					"pseudoClassLocks":        []any{},
					"mutationBreakpoints":     map[string]any{},
					"traits":                  map[string]any{},
					"browsingContextID":       1,
					"isDisplayed":             true,
					"parent":                  nil,
				}
				var ancestors []any
				switch nodeActor {
				case "plex.conn0.node2": // body → ancestors: [html, document]
					ancestors = []any{htmlForm, docForm}
				case "plex.conn0.nodeHtml1": // html → ancestors: [document]
					ancestors = []any{docForm}
				default:
					ancestors = []any{}
				}
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"nodes": ancestors,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.thread1":
			switch message["type"].(string) {
			case "attach":
				if err := writePacket(logger, writer, map[string]any{
					"from":         message["to"],
					"type":         "paused",
					"actor":        "plex.conn0.pause1",
					"poppedFrames": []any{},
					"why":          map[string]any{"type": "attached"},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "resume":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"type": "resumed",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "detach":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"type": "detached",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.console1":
			switch message["type"].(string) {
			case "startListeners":
				if err := writePacket(logger, writer, map[string]any{
					"from":             message["to"],
					"startedListeners": message["listeners"],
					"nativeConsoleAPI": true,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getCachedMessages":
				if err := writePacket(logger, writer, map[string]any{
					"from":     message["to"],
					"messages": []any{},
					// { "_type": "PageError", "message": "...", "timeStamp": 0, ... }
					// { "_type": "ConsoleAPI", "message": "...", "timeStamp": 0, ... }
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.cssProperties1":

			switch message["type"].(string) {
			case "getCSSDatabase":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"properties": map[string]any{
						"color": map[string]any{
							"isInherited":   true,
							"values":        []string{"currentcolor", "transparent", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"color"},
						},
						"display": map[string]any{
							"isInherited":   false,
							"values":        []string{"block", "inline", "inline-block", "flex", "grid", "none", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"display"},
						},
						"background": map[string]any{
							"isInherited":   false,
							"values":        []string{"none", "transparent", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"background-color", "background-image", "background-repeat", "background-position", "background-size"},
						},
						"background-color": map[string]any{
							"isInherited":   false,
							"values":        []string{"transparent", "currentcolor", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"background-color"},
						},
						"font-size": map[string]any{
							"isInherited":   true,
							"values":        []string{"medium", "small", "large", "x-small", "x-large", "smaller", "larger", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"font-size"},
						},
						"font-weight": map[string]any{
							"isInherited":   true,
							"values":        []string{"normal", "bold", "bolder", "lighter", "100", "200", "300", "400", "500", "600", "700", "800", "900", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"font-weight"},
						},
						"margin": map[string]any{
							"isInherited":   false,
							"values":        []string{"auto", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"margin-top", "margin-right", "margin-bottom", "margin-left"},
						},
						"padding": map[string]any{
							"isInherited":   false,
							"values":        []string{"inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"padding-top", "padding-right", "padding-bottom", "padding-left"},
						},
						"width": map[string]any{
							"isInherited":   false,
							"values":        []string{"auto", "max-content", "min-content", "fit-content", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"width"},
						},
						"height": map[string]any{
							"isInherited":   false,
							"values":        []string{"auto", "max-content", "min-content", "fit-content", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"height"},
						},
						"position": map[string]any{
							"isInherited":   false,
							"values":        []string{"static", "relative", "absolute", "fixed", "sticky", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"position"},
						},
						"border": map[string]any{
							"isInherited":   false,
							"values":        []string{"none", "inherit", "initial", "unset"},
							"supports":      []string{},
							"subproperties": []string{"border-width", "border-style", "border-color"},
						},
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
				// unrecognized packet error
			}
		case "plex.conn0.accessibility1":
			switch message["type"].(string) {
			case "bootstrap":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"state": map[string]any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getTraits":
				if err := writePacket(logger, writer, map[string]any{
					"from":   message["to"],
					"traits": map[string]any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getWalker":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"walker": map[string]any{
						"actor": "plex.conn0.accessibilityWalker1",
					},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			case "getSimulator":
				if err := writePacket(logger, writer, map[string]any{
					"from":      message["to"],
					"simulator": nil,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.accessibilityWalker1":
			switch message["type"].(string) {
			case "children":
				if err := writePacket(logger, writer, map[string]any{
					"from":     message["to"],
					"children": []any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.layout1":
			switch message["type"].(string) {
			case "getGrids":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"grids": []any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getCurrentFlexbox":
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"flexbox": nil,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getCurrentGrid":
				if err := writePacket(logger, writer, map[string]any{
					"from": message["to"],
					"grid": nil,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}
		case "plex.conn0.reflow1":
			switch message["type"].(string) {
			case "start", "stop":
				// no response

			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}

			}
		case "plex.conn0.pageStyle1":
			switch message["type"].(string) {
			case "isPositionEditable":
				if err := writePacket(logger, writer, map[string]any{
					"from":  message["to"],
					"value": false,
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getApplied":
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"entries": []any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getComputed":
				if err := writePacket(logger, writer, map[string]any{
					"from":      message["to"],
					"computed":  map[string]any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			case "getLayout":
				if err := writePacket(logger, writer, map[string]any{
					"from":        message["to"],
					"width":       0,
					"height":      0,
					"autoMargins": map[string]any{},
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			default:
				logger.DebugContext(ctx, "unhandled packet")
				if err := writePacket(logger, writer, map[string]any{
					"from":    message["to"],
					"error":   "unrecognizedPacketType",
					"message": "unable to process request",
				}); err != nil {
					logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
					return
				}
			}

		case "plex.conn0.processDescriptor1", "plex.conn0.networkParent1":
			fallthrough

		default:
			logger.DebugContext(ctx, "unhandled packet")
			if err := writePacket(logger, writer, map[string]any{
				"from":    message["to"],
				"error":   "unrecognizedPacketType",
				"message": "unable to process request",
			}); err != nil {
				logger.ErrorContext(ctx, "packet write error", slog.String("error", err.Error()))
				return
			}

		}

	}
}
