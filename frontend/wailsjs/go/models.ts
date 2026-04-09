export namespace service {
	
	export class LLMPrepareRequest {
	    audience: string;
	    title: string;
	    bibleText: string;
	    hymn: string;
	    preacher: string;
	    churchName: string;
	    sermonDate: string;
	    sourceUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMPrepareRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.audience = source["audience"];
	        this.title = source["title"];
	        this.bibleText = source["bibleText"];
	        this.hymn = source["hymn"];
	        this.preacher = source["preacher"];
	        this.churchName = source["churchName"];
	        this.sermonDate = source["sermonDate"];
	        this.sourceUrl = source["sourceUrl"];
	    }
	}
	export class LLMPrepareResult {
	    success: boolean;
	    message: string;
	    status: string;
	    jsonFile: string;
	    jsonText?: string;
	    steps: string[];
	
	    static createFrom(source: any = {}) {
	        return new LLMPrepareResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.status = source["status"];
	        this.jsonFile = source["jsonFile"];
	        this.jsonText = source["jsonText"];
	        this.steps = source["steps"];
	    }
	}
	export class PNGGenerateResult {
	    success: boolean;
	    message: string;
	    htmlFile: string;
	    pngFile: string;
	    dpi: number;
	    widthPx: number;
	    heightPx: number;
	
	    static createFrom(source: any = {}) {
	        return new PNGGenerateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.htmlFile = source["htmlFile"];
	        this.pngFile = source["pngFile"];
	        this.dpi = source["dpi"];
	        this.widthPx = source["widthPx"];
	        this.heightPx = source["heightPx"];
	    }
	}
	export class QTStep2Data {
	    audience: string;
	    title: string;
	    bibleText: string;
	    hymn: string;
	    preacher: string;
	    churchName: string;
	    sermonDate: string;
	    sourceURL: string;
	    summaryTitle: string;
	    summaryBody: string;
	    messageTitle1: string;
	    messageBody1: string;
	    messageTitle2: string;
	    messageBody2: string;
	    messageTitle3: string;
	    messageBody3: string;
	    reflectionItem1: string;
	    reflectionItem2: string;
	    reflectionItem3: string;
	    prayerTitle: string;
	    prayerBody: string;
	
	    static createFrom(source: any = {}) {
	        return new QTStep2Data(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.audience = source["audience"];
	        this.title = source["title"];
	        this.bibleText = source["bibleText"];
	        this.hymn = source["hymn"];
	        this.preacher = source["preacher"];
	        this.churchName = source["churchName"];
	        this.sermonDate = source["sermonDate"];
	        this.sourceURL = source["sourceURL"];
	        this.summaryTitle = source["summaryTitle"];
	        this.summaryBody = source["summaryBody"];
	        this.messageTitle1 = source["messageTitle1"];
	        this.messageBody1 = source["messageBody1"];
	        this.messageTitle2 = source["messageTitle2"];
	        this.messageBody2 = source["messageBody2"];
	        this.messageTitle3 = source["messageTitle3"];
	        this.messageBody3 = source["messageBody3"];
	        this.reflectionItem1 = source["reflectionItem1"];
	        this.reflectionItem2 = source["reflectionItem2"];
	        this.reflectionItem3 = source["reflectionItem3"];
	        this.prayerTitle = source["prayerTitle"];
	        this.prayerBody = source["prayerBody"];
	    }
	}
	export class QTStep2PreviewResult {
	    success: boolean;
	    message: string;
	    htmlFile: string;
	
	    static createFrom(source: any = {}) {
	        return new QTStep2PreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.htmlFile = source["htmlFile"];
	    }
	}
	export class QTStep3FileResult {
	    success: boolean;
	    status: string;
	    filePath?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new QTStep3FileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.status = source["status"];
	        this.filePath = source["filePath"];
	        this.error = source["error"];
	    }
	}
	export class QTStep3Request {
	    makeHtml: boolean;
	    makePdf: boolean;
	    makeDocx: boolean;
	    makePptx: boolean;
	    makePng: boolean;
	    dpi: number;
	
	    static createFrom(source: any = {}) {
	        return new QTStep3Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.makeHtml = source["makeHtml"];
	        this.makePdf = source["makePdf"];
	        this.makeDocx = source["makeDocx"];
	        this.makePptx = source["makePptx"];
	        this.makePng = source["makePng"];
	        this.dpi = source["dpi"];
	    }
	}
	export class QTStep3Result {
	    html: QTStep3FileResult;
	    pdf: QTStep3FileResult;
	    docx: QTStep3FileResult;
	    pptx: QTStep3FileResult;
	    png: QTStep3FileResult;
	
	    static createFrom(source: any = {}) {
	        return new QTStep3Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.html = this.convertValues(source["html"], QTStep3FileResult);
	        this.pdf = this.convertValues(source["pdf"], QTStep3FileResult);
	        this.docx = this.convertValues(source["docx"], QTStep3FileResult);
	        this.pptx = this.convertValues(source["pptx"], QTStep3FileResult);
	        this.png = this.convertValues(source["png"], QTStep3FileResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SourcePrepareRequest {
	    sourceType: string;
	    inputMode: string;
	    sourceUrl: string;
	    sourcePath: string;
	    textContent: string;
	
	    static createFrom(source: any = {}) {
	        return new SourcePrepareRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceType = source["sourceType"];
	        this.inputMode = source["inputMode"];
	        this.sourceUrl = source["sourceUrl"];
	        this.sourcePath = source["sourcePath"];
	        this.textContent = source["textContent"];
	    }
	}
	export class SourcePrepareResult {
	    success: boolean;
	    message: string;
	    status: string;
	    sourceType: string;
	    rawText: string;
	    txtFile: string;
	    steps: string[];
	
	    static createFrom(source: any = {}) {
	        return new SourcePrepareResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.status = source["status"];
	        this.sourceType = source["sourceType"];
	        this.rawText = source["rawText"];
	        this.txtFile = source["txtFile"];
	        this.steps = source["steps"];
	    }
	}
	export class VideoMeta {
	    title: string;
	    uploader: string;
	    channel: string;
	    thumbnail: string;
	    description: string;
	    webpageUrl: string;
	    uploadDate: string;
	    uploadDateText: string;
	    duration: number;
	    durationText: string;
	
	    static createFrom(source: any = {}) {
	        return new VideoMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.uploader = source["uploader"];
	        this.channel = source["channel"];
	        this.thumbnail = source["thumbnail"];
	        this.description = source["description"];
	        this.webpageUrl = source["webpageUrl"];
	        this.uploadDate = source["uploadDate"];
	        this.uploadDateText = source["uploadDateText"];
	        this.duration = source["duration"];
	        this.durationText = source["durationText"];
	    }
	}

}

